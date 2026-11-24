// scenario-admin serves a web admin of the story: the graph of what the player can reach from a start node up to the
// first battle, with each area drawn from its scene map (internal/maplight) and each event, exit and person one click
// away, and the checks of internal/scenegraph.
//
//	go run ./cmd/scenario-admin [-addr :8080] [-scenario hero] [-root .]   then open http://localhost:8080
package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"srpg/internal/content"
	"srpg/internal/maplight"
	"srpg/internal/scenegraph"
	"srpg/internal/scenemap"
	"srpg/internal/sprite"
)

//go:embed web
var web embed.FS

type server struct {
	root, scenario string

	mu     sync.RWMutex // guards the loaded data, the atlas and the caches
	data   *content.Data
	atlas  *maplight.Atlas
	maps   map[string]*scenemap.Map
	pngs   map[string][]byte
	sheets map[string]*sprite.Sheet

	render sync.Mutex // one render at a time: a render holds a canvas of tens of megabytes
}

func (s *server) load() error {
	d, err := content.LoadScenario(filepath.Join(s.root, "assets", "data"), s.scenario)
	if err != nil {
		return err
	}
	a, err := maplight.LoadAtlas(filepath.Join(s.root, "assets", "graphics"))
	if err != nil {
		return err
	}
	sheets, err := sprite.LoadDir(filepath.Join(s.root, "assets", "sprites"))
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.data, s.atlas, s.maps, s.pngs, s.sheets = d, a, map[string]*scenemap.Map{}, map[string][]byte{}, sheets
	s.mu.Unlock()
	return nil
}

// sceneMap reads a scene map by id (assets/graphics/maps/<id>.json), nil when there is none.
func (s *server) sceneMap(id string) *scenemap.Map {
	s.mu.RLock()
	m, ok := s.maps[id]
	s.mu.RUnlock()
	if ok {
		return m
	}
	b, err := os.ReadFile(filepath.Join(s.root, "assets", "graphics", "maps", id+".json"))
	if err == nil {
		var parsed bool
		if m, parsed, err = scenemap.Parse(b); err != nil || !parsed {
			m = nil
		}
	}
	s.mu.Lock()
	s.maps[id] = m
	s.mu.Unlock()
	return m
}

// current is the loaded data and atlas; a reload swaps them, so a request keeps the pair it began with.
func (s *server) current() (*content.Data, *maplight.Atlas) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data, s.atlas
}

func (s *server) env(a *maplight.Atlas) scenegraph.Env {
	s.mu.RLock()
	sheets := s.sheets
	s.mu.RUnlock()
	return scenegraph.Env{Map: s.sceneMap, Catalog: a.Catalog, HasSheet: func(id string) bool { return sheets[id] != nil },
		Portrait: func(name string) bool {
			_, err := os.Stat(filepath.Join(s.root, "assets", "graphics", "portraits", name+".png"))
			return err == nil
		}}
}

// graph builds the graph from ?start= through ?battles= battles (default 1). ?assume=auto takes the flags the story
// has set before the start as set; a comma-separated list names them.
func (s *server) graph(c *gin.Context) {
	d, a := s.current()
	start := c.DefaultQuery("start", d.Campaign.Start)
	opt := scenegraph.Options{Battles: 1}
	if n, err := strconv.Atoi(c.Query("battles")); err == nil && n > 0 {
		opt.Battles = n
	}
	switch as := c.Query("assume"); as {
	case "":
	case "auto":
		opt.Assume = scenegraph.AssumeBefore(d, start)
	default:
		opt.Assume = strings.Split(as, ",")
	}
	c.JSON(http.StatusOK, scenegraph.Build(d, s.env(a), start, opt))
}

func (s *server) outline(c *gin.Context) {
	d, _ := s.current()
	c.JSON(http.StatusOK, gin.H{"start": d.Campaign.Start, "chapters": scenegraph.Outline(d)})
}

var stem = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)

// portrait serves graphics/portraits/<name>.png.
func (s *server) portrait(c *gin.Context) {
	name := c.Query("name")
	if !stem.MatchString(name) {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "max-age=300")
	c.File(filepath.Join(s.root, "assets", "graphics", "portraits", name+".png"))
}

// spritePNG is the first idle frame of a sprite sheet facing the viewer, trimmed and doubled, in the sheet's own
// colours.
func (s *server) spritePNG(c *gin.Context) {
	s.mu.RLock()
	sh := s.sheets[c.Query("id")]
	s.mu.RUnlock()
	if sh == nil {
		c.Status(http.StatusNotFound)
		return
	}
	sheet := sh.Render(nil)
	row := 0
	for _, want := range []string{"S", "SE", "SW"} {
		if i := slices.Index(sh.Rows, want); i >= 0 {
			row = i
			break
		}
	}
	col := sh.Animations["idle"].FirstColumn
	w, h := sh.Cell[0], sh.Cell[1]
	cell := sheet.SubImage(image.Rect(col*w, row*h, (col+1)*w, (row+1)*h)).(*image.RGBA)
	box := image.Rectangle{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if cell.RGBAAt(cell.Rect.Min.X+x, cell.Rect.Min.Y+y).A > 0 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if box.Empty() {
		c.Status(http.StatusNotFound)
		return
	}
	out := image.NewRGBA(image.Rect(0, 0, box.Dx()*2, box.Dy()*2))
	for y := 0; y < out.Rect.Dy(); y++ {
		for x := 0; x < out.Rect.Dx(); x++ {
			out.SetRGBA(x, y, cell.RGBAAt(cell.Rect.Min.X+box.Min.X+x/2, cell.Rect.Min.Y+box.Min.Y+y/2))
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, out)
	c.Header("Cache-Control", "max-age=300")
	c.Data(http.StatusOK, "image/png", buf.Bytes())
}

func (s *server) vertex(c *gin.Context) {
	d, a := s.current()
	v, err := scenegraph.Detail(d, s.env(a), c.Query("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, v)
}

// mapPNG draws the whole canvas of the scene map of an area vertex ("area:explore01/zhuo"), once.
func (s *server) mapPNG(c *gin.Context) {
	id := c.Query("id")
	file, area, ok := strings.Cut(strings.TrimPrefix(id, "area:"), "/")
	d, atlas := s.current()
	x := d.Explores[file]
	if !ok || x == nil || x.Areas[area] == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no area " + id})
		return
	}
	a := x.Areas[area]
	s.mu.RLock()
	b, cached := s.pngs[id]
	s.mu.RUnlock()
	if !cached {
		m := s.sceneMap(a.Map)
		if m == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no map " + a.Map})
			return
		}
		if a.Grass != nil {
			cp := *m
			cp.Grass = a.Grass
			m = &cp
		}
		s.render.Lock()
		img, _, err := atlas.Render(m)
		s.render.Unlock()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		b = buf.Bytes()
		s.mu.Lock()
		s.pngs[id] = b
		s.mu.Unlock()
	}
	c.Data(http.StatusOK, "image/png", b)
}

func (s *server) reload(c *gin.Context) {
	if err := s.load(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	scenario := flag.String("scenario", "hero", "scenario id")
	root := flag.String("root", ".", "ver5 directory")
	flag.Parse()
	s := &server{root: *root, scenario: *scenario}
	if err := s.load(); err != nil {
		log.Fatal(err)
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	api := r.Group("/api")
	api.GET("/graph", s.graph)
	api.GET("/vertex", s.vertex)
	api.GET("/outline", s.outline)
	api.GET("/portrait", s.portrait)
	api.GET("/sprite", s.spritePNG)
	api.GET("/map.png", s.mapPNG)
	api.POST("/reload", s.reload)
	r.GET("/", func(c *gin.Context) {
		b, _ := web.ReadFile("web/index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", b)
	})
	fmt.Printf("scenario admin: http://%s\n", *addr)
	log.Fatal(r.Run(*addr))
}
