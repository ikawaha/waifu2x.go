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

	var pix *Pixels
	switch t := img.(type) {
	case *image.RGBA:
		pix = &Pixels{
			Width:  img.Bounds().Max.X,
			Height: img.Bounds().Max.Y,
			Pix:    t.Pix,
		}
	case *image.NRGBA:
		pix = &Pixels{
			Width:  img.Bounds().Max.X,
			Height: img.Bounds().Max.Y,
			Pix:    t.Pix,
		}
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

	scale := 2.0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		R, G, B, A, _ := pix.Decompose()
		R, G, B = noise.Encode(R, G, B)

		var err error
		if R, err = R.NewExtendPixels(scale); err != nil {
			b.Fatalf("unexpected extend image error, R:%v", err)
		}
		if G, err = G.NewExtendPixels(scale); err != nil {
			b.Fatalf("unexpected extend image error, G:%v", err)
		}
		if B, err = B.NewExtendPixels(scale); err != nil {
			b.Fatalf("unexpected extend image error, B:%v", err)
		}
		if A, err = A.NewExtendPixels(scale); err != nil {
			b.Fatalf("unexpected extend image error, A:%v", err)
		}

		R, G, B = model2x.Encode(R, G, B)
		if _, err := Compose(R, G, B, A); err != nil {
			b.Fatalf("unexpected compose error,%v", err)
		}
	}

}
