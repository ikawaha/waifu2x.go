package waifu2x

import (
	"image"
	"image/png"
	"os"
	"testing"
)

func Benchmark01(b *testing.B) {
	const (
		scale2xModelFile = "models/anime_style_art_rgb/scale2.0x_model.json"
		noiseModelFile   = "models/anime_style_art_rgb/noise2_model.json"
	)

	fp, err := os.Open("testdata/neko_small.png")
	if err != nil {
		b.Fatalf("input file %v, %v", "testdata/neko.png", err)
	}
	defer fp.Close()
	img, err := png.Decode(fp)
	if err != nil {
		b.Fatalf("load file %v, %v", "testdata/neko.png", err)
	}

	var pix []uint8
	switch t := img.(type) {
	case *image.RGBA:
		pix = t.Pix
	case *image.NRGBA:
		pix = t.Pix
	default:
		b.Fatalf("unknown image format, %T", t)
	}

	model2x, err := LoadModelFile(scale2xModelFile)
	if err != nil {
		b.Fatalf("load scale2x model, %v", err)
	}

	noise, err := LoadModelFile(noiseModelFile)
	if err != nil {
		b.Fatalf("load noise model, %v", err)
	}

	model := Waifu2x{
		Scale2xModel: model2x,
		NoiseModel:   noise,
		Scale:        2.0,
		IsDenoising:  true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Calc(pix, img.Bounds().Max.X, img.Bounds().Max.Y)
	}

}
