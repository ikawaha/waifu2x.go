package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"

	"github.com/ikawaha/waifu2x-go"
	"github.com/ikawaha/waifu2x-go/data"
)

const (
	scale2xModelFile = "models/anime_style_art_rgb/scale2.0x_model.json"
	noiseModelFile   = "models/anime_style_art_rgb/noise2_model.json"
)

func run(opt *option) error {
	fp, err := os.Open(opt.input)
	if err != nil {
		return fmt.Errorf("input file %v, %v", opt.input, err)
	}
	defer fp.Close()
	img, err := png.Decode(fp)
	if err != nil {
		return fmt.Errorf("load file %v, %v", opt.input, err)
	}

	var pix []uint8
	switch t := img.(type) {
	case *image.RGBA:
		pix = t.Pix
	case *image.NRGBA:
		pix = t.Pix
	default:
		return fmt.Errorf("unknown image format, %T", t)
	}

	buf0, err := data.Asset(scale2xModelFile)
	if err != nil {
		return fmt.Errorf("open scale2x model, %v", err)
	}
	model2x, err := waifu2x.LoadModel(bytes.NewBuffer(buf0))
	if err != nil {
		return fmt.Errorf("load scale2x model, %v", err)
	}

	buf1, err := data.Asset(noiseModelFile)
	if err != nil {
		return fmt.Errorf("open noise model, %v", err)
	}
	noise, err := waifu2x.LoadModel(bytes.NewBuffer(buf1))
	if err != nil {
		return fmt.Errorf("load noise model, %v", err)
	}

	model := waifu2x.Waifu2x{
		Scale2xModel: model2x,
		NoiseModel:   noise,
		Scale:        opt.scale,
		IsDenoising:  true,
	}

	pix, width, height := model.Calc(pix, img.Bounds().Max.X, img.Bounds().Max.Y)
	rect0 := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: width, Y: height},
	}

	switch t := img.(type) {
	case *image.RGBA:
		t.Pix = pix
		t.Rect = rect0
		t.Stride = rect0.Dx() * 4
	case *image.NRGBA:
		t.Pix = pix
		t.Rect = rect0
		t.Stride = rect0.Dx() * 4
	default:
		return fmt.Errorf("unknown image format, %T", t)
	}

	var w io.Writer = os.Stdout
	if opt.output != "" {
		fp, err := os.Create(opt.output)
		if err != nil {
			return fmt.Errorf("output file, %v", err)
		}
		defer fp.Close()
		w = fp
	}
	if err := png.Encode(w, img); err != nil {
		panic(err)
	}
	return nil
}
