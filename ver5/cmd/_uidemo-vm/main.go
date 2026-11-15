// Temporary tool: plays a cmd/_uidemo demo headlessly through Ebitengine's exp/vmhost and
// captures it, with no visible window. The demo runs as a vmhost guest (built with the
// ebitenginevmguest tag: no window, no GPU); this host replays its frames on a hidden window,
// one tick per frame, faster than real time. Chosen frames are saved as PNG and every frame
// can go to a 60 fps MP4 through ffmpeg.
//
//	go run ./cmd/_uidemo-vm -ui turn -frames 240 -mp4 reports/uidemo/turn.mp4
//	go run ./cmd/_uidemo-vm -ui damage -frames 380 -at 39,51,69 -out reports/uidemo/damage
//	go run ./cmd/_uidemo-vm -ui duel -frames 420 -every 30 -mp4 reports/uidemo/duel.mp4
//	go run ./cmd/_uidemo-vm -ui uiscale -frames 760 -size 2400x1080 -uiscale 160 -at 380,720
//	go run ./cmd/_uidemo-vm -ui uiscale -frames 760 -sheet -at 260,380,720
//	go run ./cmd/_uidemo-vm -ui zz-tmp -frames 30000 -still 3 -mp4 reports/uidemo/battle.mp4
//
// -sheet plays the demo once per reference screen (phone landscape at 130% and 160%, a 4:3
// tablet, the desktop window; see docs_old/plans/PLAN_ui-scale-ppi.md) and puts each -at frame
// of all four side by side in <out>/sheet-NNNN.png.
//
// The host still opens a (hidden) GLFW window, so the display must be awake: wrap the run in
// `caffeinate -d -i` when the machine may sleep.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

var (
	ui     = flag.String("ui", "", "demo to play (see cmd/_uidemo)")
	frames = flag.Int("frames", 120, "frames to play, one game tick each (60 per second)")
	stage  = flag.String("stage", "", "battle stage the demo plays (default: the first)")
	duel   = flag.String("duel", "", "stage duel the duel demo plays (default: the first)")
	node   = flag.String("node", "", "story node the scene demos jump to")
	out    = flag.String("out", "", "PNG directory (default reports/uidemo/<ui>)")
	every  = flag.Int("every", 0, "save every n-th frame as PNG; 0 for none")
	at     = flag.String("at", "", "comma-separated frames to save as PNG, e.g. 30,96,264")
	mp4    = flag.String("mp4", "", "write every frame to this MP4 (needs ffmpeg)")
	crf    = flag.Int("crf", 18, "x264 quality for -mp4: 0 is lossless, 18 is visually lossless")
	half   = flag.Bool("half", false, "write the MP4 at half size")
	size   = flag.String("size", "1280x850", "screen size in device pixels, e.g. 2400x1080 for a phone")
	uiPct  = flag.Int("uiscale", 0, "UI size setting in percent (80-160); 0 keeps the default")
	sheet  = flag.Bool("sheet", false, "capture the -at frames on every reference screen into one sheet each")
	prof   = flag.String("cpuprofile", "", "write CPU profiles to <this>.guest.pprof (the game) and <this>.host.pprof (the capture)")
	timing = flag.Bool("timing", false, "print how long each step of the run took")
	still  = flag.Float64("still", 0, "stop once the screen has not changed for this many seconds; 0 plays every frame")
)

// screens are the reference screens -sheet plays on.
var screens = []struct {
	name    string
	w, h    int
	uiScale int
}{
	{"phone 130%", 2400, 1080, 130},
	{"phone 160%", 2400, 1080, 160},
	{"tablet 4:3 130%", 2048, 1536, 130},
	{"desktop 100%", 1280, 850, 100},
}

// host drives the demo guest from a hidden window: each frame it runs one guest tick, has
// the guest render, composites the result into screen and hands the pixels to the sinks.
type host struct {
	guest   *vmhost.GuestSession
	w, h    int
	screen  *ebiten.Image
	pixels  []byte
	pngAt   map[int]bool
	video   io.WriteCloser
	writers sync.WaitGroup
	frame   int
	err     error
	last    []byte // the previous frame's pixels, for -still
	same    int    // frames in a row equal to last
}

func (h *host) Update() error {
	if h.screen == nil {
		h.screen = ebiten.NewImage(h.w, h.h)
		h.pixels = make([]byte, 4*h.w*h.h)
		if err := h.guest.SetOutsideScreen(h.screen); err != nil {
			return err
		}
	}
	// a batch of frames per host frame keeps the run fast; each still gets its own tick
	for range 8 {
		if h.frame >= *frames {
			return ebiten.Termination
		}
		h.guest.AdvanceTicks(1)
		h.guest.AdvanceFrame()
		if !h.guest.WaitFrame() {
			return ebiten.Termination // the guest ended (its demo ran out) or failed; see Err
		}
		if !h.guest.CompositeFrame() {
			return errors.New("compositing the guest frame failed")
		}
		h.screen.ReadPixels(h.pixels)
		if *still > 0 {
			if bytes.Equal(h.pixels, h.last) {
				if h.same++; h.same >= int(*still*60) {
					return ebiten.Termination
				}
			} else {
				h.same, h.last = 0, append(h.last[:0], h.pixels...)
			}
		}
		if h.video != nil {
			if _, err := h.video.Write(h.pixels); err != nil {
				return fmt.Errorf("writing frame %d to ffmpeg: %w", h.frame, err)
			}
		}
		if h.pngAt[h.frame] || *every > 0 && h.frame%*every == 0 {
			h.savePNG(h.frame, append([]byte(nil), h.pixels...))
		}
		h.frame++
	}
	return nil
}

// savePNG encodes a frame in the background; PNG compression is the slow part.
func (h *host) savePNG(frame int, pix []byte) {
	h.writers.Add(1)
	go func() {
		defer h.writers.Done()
		img := &image.RGBA{Pix: pix, Stride: 4 * h.w, Rect: image.Rect(0, 0, h.w, h.h)}
		f, err := os.Create(filepath.Join(*out, fmt.Sprintf("frame-%04d.png", frame)))
		if err == nil {
			err = png.Encode(f, img)
			f.Close()
		}
		if err != nil {
			h.err = err
		}
	}()
}

func (h *host) Draw(*ebiten.Image) {}

func (h *host) Layout(w, h2 int) (int, int) { return w, h2 }

// atFrames are the frame numbers in -at.
func atFrames() []int {
	var out []int
	for _, s := range strings.Split(*at, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

var started = time.Now()

// step prints how far into the run a step ended, with -timing.
func step(name string) {
	if *timing {
		fmt.Fprintf(os.Stderr, "[%5.2fs] %s\n", time.Since(started).Seconds(), name)
	}
}

func main() {
	err := run()
	step("exit")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()
	if *ui == "" {
		return errors.New("choose a demo with -ui")
	}
	if *out == "" {
		*out = filepath.Join("reports", "uidemo", *ui)
	}
	if *sheet {
		return runSheet()
	}
	h := &host{pngAt: map[int]bool{}}
	if _, err := fmt.Sscanf(*size, "%dx%d", &h.w, &h.h); err != nil {
		return fmt.Errorf("bad -size %q: %w", *size, err)
	}
	for _, n := range atFrames() {
		h.pngAt[n] = true
	}
	if len(h.pngAt) > 0 || *every > 0 {
		if err := os.MkdirAll(*out, 0755); err != nil {
			return err
		}
	}

	// a short temp dir keeps the unix socket path within macOS's limit
	dir, err := os.MkdirTemp("", "uv")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	ln, err := net.Listen("unix", filepath.Join(dir, "g.sock"))
	if err != nil {
		return err
	}
	defer ln.Close()
	endpoint, err := vmhost.EndpointURLFromAddr(ln.Addr())
	if err != nil {
		return err
	}

	// the guest is the ordinary demo command, built so that RunGame connects here
	guestBin := filepath.Join(dir, "guest")
	build := exec.Command("go", "build", "-tags", "ebitenginevmguest", "-o", guestBin, "./cmd/_uidemo")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("building the guest: %w", err)
	}
	step("guest built")
	// one more frame than captured: the guest stops itself after its last frame
	guestCmd := exec.Command(guestBin, "-ui", *ui, "-frames", strconv.Itoa(*frames+1), "-every", "0", "-stage", *stage, "-duel", *duel, "-node", *node,
		"-size", *size, "-uiscale", strconv.Itoa(*uiPct))
	if *prof != "" {
		guestCmd.Args = append(guestCmd.Args, "-cpuprofile", *prof+".guest.pprof")
	}
	guestCmd.Env = append(os.Environ(), "EBITENGINE_VM_ENDPOINT="+endpoint)
	guestCmd.Stdout, guestCmd.Stderr = os.Stderr, os.Stderr
	if err := guestCmd.Start(); err != nil {
		return fmt.Errorf("starting the guest: %w", err)
	}
	step("guest started")
	defer func() {
		// the guest ends itself after its last frame; give it time to flush its profile before killing it
		exited := make(chan struct{})
		go func() { _ = guestCmd.Wait(); close(exited) }()
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = guestCmd.Process.Kill()
			<-exited
		}
	}()

	if dl, ok := ln.(interface{ SetDeadline(time.Time) error }); ok {
		_ = dl.SetDeadline(time.Now().Add(60 * time.Second)) // loading the game data takes a while
	}
	conn, err := ln.Accept()
	step("guest connected")
	if err != nil {
		return fmt.Errorf("waiting for the guest: %w", err)
	}
	_ = ln.Close()
	h.guest, err = vmhost.NewGuestSession(conn, &vmhost.NewGuestSessionOptions{IdleTimeout: 60 * time.Second})
	if err != nil {
		return err
	}

	var ffmpeg *exec.Cmd
	if *mp4 != "" {
		if err := os.MkdirAll(filepath.Dir(*mp4), 0755); err != nil {
			return err
		}
		args := []string{"-y", "-loglevel", "error", "-f", "rawvideo", "-pix_fmt", "rgba",
			"-s", fmt.Sprintf("%dx%d", h.w, h.h), "-r", "60", "-i", "-"}
		if *half {
			args = append(args, "-vf", "scale=iw/2:-2:flags=area")
		}
		pix := "yuv420p"
		if *crf == 0 {
			pix = "yuv444p" // keep full colour detail when lossless
		}
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(*crf), "-pix_fmt", pix, "-movflags", "+faststart", *mp4)
		ffmpeg = exec.Command("ffmpeg", args...)
		ffmpeg.Stderr = os.Stderr
		if h.video, err = ffmpeg.StdinPipe(); err != nil {
			return err
		}
		if err := ffmpeg.Start(); err != nil {
			return fmt.Errorf("starting ffmpeg: %w", err)
		}
	}

	if *prof != "" {
		f, err := os.Create(*prof + ".host.pprof")
		if err != nil {
			return err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}
	start := time.Now()
	ebiten.SetWindowVisible(false)
	step("host window")
	runErr := ebiten.RunGame(h)
	step("frames played")
	_ = h.guest.Close()
	h.writers.Wait()
	step("pngs written")
	if h.video != nil {
		h.video.Close()
		if err := ffmpeg.Wait(); err != nil {
			return fmt.Errorf("ffmpeg: %w", err)
		}
	}
	if runErr != nil {
		return runErr
	}
	if err := h.guest.Err(); err != nil && !errors.Is(err, ebiten.Termination) {
		return fmt.Errorf("guest: %w", err)
	}
	if h.err != nil {
		return h.err
	}
	fmt.Fprintf(os.Stderr, "%d frames in %.1fs", h.frame, time.Since(start).Seconds())
	if *mp4 != "" {
		fmt.Fprintf(os.Stderr, ", video %s", *mp4)
	}
	fmt.Fprintln(os.Stderr)
	return nil
}
