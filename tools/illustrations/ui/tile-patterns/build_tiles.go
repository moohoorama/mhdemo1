package main

import (
	"image"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
)

const blendWidth = 32

var contrast = map[string]float64{
	"ruyi-cloud": 0.56,
	"leiwen":     0.17,
	"shou":       0.56,
}

func main() {
	for _, name := range []string{"ruyi-cloud", "leiwen", "shou"} {
		if err := build(name); err != nil {
			log.Fatal(err)
		}
	}
}

func build(name string) error {
	input, err := os.Open(name + ".png")
	if err != nil {
		return err
	}
	defer input.Close()
	source, err := png.Decode(input)
	if err != nil {
		return err
	}
	bounds := source.Bounds()
	var imageOut *image.RGBA
	if name == "shou" {
		imageOut = mirroredShou(source)
	} else {
		imageOut = image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(imageOut, imageOut.Bounds(), source, bounds.Min, draw.Src)
		w, h := bounds.Dx(), bounds.Dy()
		if w <= blendWidth*2 || h <= blendWidth*2 {
			return os.ErrInvalid
		}
		for y := range h {
			for d := range blendWidth {
				mix(imageOut, d, y, w-1-d, y, d)
			}
		}
		for x := range w {
			for d := range blendWidth {
				mix(imageOut, x, d, x, h-1-d, d)
			}
		}
	}
	var targetMean []float64
	if name == "leiwen" {
		targetMean = []float64{30, 43, 42}
	}
	softenContrast(imageOut, contrast[name], targetMean)

	output, err := os.Create(name + "-tile.png")
	if err != nil {
		return err
	}
	defer output.Close()
	return png.Encode(output, imageOut)
}

func mirroredShou(source image.Image) *image.RGBA {
	const halfW, halfH = 314, 315
	output := image.NewRGBA(image.Rect(0, 0, halfW*2, halfH*2))
	for y := range halfH * 2 {
		sy := y
		if sy >= halfH {
			sy = halfH*2 - 1 - sy
		}
		for x := range halfW * 2 {
			sx := x
			if sx >= halfW {
				sx = halfW*2 - 1 - sx
			}
			output.Set(x, y, source.At(sx, 288+sy))
		}
	}
	return output
}

func softenContrast(img *image.RGBA, strength float64, targetMean []float64) {
	var sum [3]float64
	count := float64(img.Bounds().Dx() * img.Bounds().Dy())
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			index := img.PixOffset(x, y)
			for channel := range 3 {
				sum[channel] += float64(img.Pix[index+channel])
			}
		}
	}
	for channel := range 3 {
		sum[channel] /= count
	}
	if targetMean == nil {
		targetMean = sum[:]
	}
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			index := img.PixOffset(x, y)
			for channel := range 3 {
				value := targetMean[channel] + (float64(img.Pix[index+channel])-sum[channel])*strength
				img.Pix[index+channel] = uint8(math.Round(value))
			}
		}
	}
}

func mix(img *image.RGBA, ax, ay, bx, by, distance int) {
	a := img.PixOffset(ax, ay)
	b := img.PixOffset(bx, by)
	weight := math.Pow(1-float64(distance)/blendWidth, 2)
	for channel := range 3 {
		left := float64(img.Pix[a+channel])
		right := float64(img.Pix[b+channel])
		middle := (left + right) / 2
		img.Pix[a+channel] = uint8(math.Round(left + (middle-left)*weight))
		img.Pix[b+channel] = uint8(math.Round(right + (middle-right)*weight))
	}
}
