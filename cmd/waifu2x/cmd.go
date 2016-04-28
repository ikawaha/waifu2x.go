package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"

	"github.com/ikawaha/waifu2x.go"
	"github.com/ikawaha/waifu2x.go/data"
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

	var pix *waifu2x.Pixels
	switch t := img.(type) {
	case *image.RGBA:
		pix = &waifu2x.Pixels{
			Width:  img.Bounds().Max.X,
			Height: img.Bounds().Max.Y,
			Pix:    t.Pix,
		}
	case *image.NRGBA:
		pix = &waifu2x.Pixels{
			Width:  img.Bounds().Max.X,
			Height: img.Bounds().Max.Y,
			Pix:    t.Pix,
		}
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

	r, g, b, a, _ := pix.Decompose()

	//XXX ノイズ適用かどうかを決めるようにする
	if true {
		r, g, b = noise.Encode(r, g, b)
	}

	if opt.scale != 1.0 {
		var err error
		if r, err = r.NewExtendPixels(opt.scale); err != nil {
			return fmt.Errorf("extend image error, %v", err)
		}
		if g, err = g.NewExtendPixels(opt.scale); err != nil {
			return fmt.Errorf("extend image error, %v", err)
		}
		if b, err = b.NewExtendPixels(opt.scale); err != nil {
			return fmt.Errorf("extend image error, %v", err)
		}
		if a, err = a.NewExtendPixels(opt.scale); err != nil {
			return fmt.Errorf("extend image error, %v", err)
		}
	}
	r, g, b = model2x.Encode(r, g, b)
	pix, err = waifu2x.Compose(r, g, b, a)
	if err != nil {
		return fmt.Errorf("compose error, %v", err)
	}

	// model := waifu2x.Waifu2x{
	// 	Scale2xModel: model2x,
	// 	NoiseModel:   noise,
	// 	Scale:        opt.scale,
	// 	IsDenoising:  true,
	// }

	// pix, width, height := model.Calc(pix, img.Bounds().Max.X, img.Bounds().Max.Y)
	rect0 := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: pix.Width, Y: pix.Height},
	}

	switch t := img.(type) {
	case *image.RGBA:
		t.Pix = pix.Pix
		t.Rect = rect0
		t.Stride = rect0.Dx() * 4
	case *image.NRGBA:
		t.Pix = pix.Pix
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
