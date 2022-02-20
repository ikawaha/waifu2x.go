package engine

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"runtime"
	"testing"
)

func TestWaifu2x_ScaleUp(t *testing.T) {
	w2x, err := NewWaifu2x(Anime, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fp, err := os.Open("../testdata/neko_small.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img, err := png.Decode(fp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	testdata := []struct {
		name  string
		scale float64
	}{
		{name: "scale up x1.0", scale: 1.0},
		{name: "scale up x1.7", scale: 1.7},
		{name: "scale up x2.0", scale: 2.0},
		{name: "scale up x3.3", scale: 3.3},
		{name: "scale up x4.0", scale: 4.0},
	}
	for _, tt := range testdata {
		t.Run(tt.name, func(t *testing.T) {
			imgX, err := w2x.ScaleUp(context.TODO(), img, tt.scale)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if want, got := int(math.Round(float64(img.Bounds().Max.X)*tt.scale)), imgX.Bounds().Max.X; want != got {
				t.Errorf("want %d, got %d", want, got)
			}
			if want, got := int(math.Round(float64(img.Bounds().Max.Y)*tt.scale)), imgX.Bounds().Max.Y; want != got {
				t.Errorf("want %d, got %d", want, got)
			}
		})
	}
}

func BenchmarkWaifu(b *testing.B) {
	tests := []struct {
		name  string
		pic   string
		mode  Mode
		noise int
		alpha bool
	}{
		{
			name:  "Neko",
			pic:   "../testdata/neko_small.png",
			mode:  Anime,
			noise: 0,
			alpha: false,
		},
		{
			name:  "Neko-alpha",
			pic:   "../testdata/neko_alpha.png",
			mode:  Anime,
			noise: 0,
			alpha: true,
		},
	}

	for _, tt := range tests {
		var rgba *image.NRGBA
		func() {
			fp, err := os.Open(tt.pic)
			if err != nil {
				b.Fatalf("failed to open the image (%s): %s", tt.pic, err)
			}
			defer fp.Close()
			img, err := png.Decode(fp)
			if err != nil {
				b.Fatalf("failed to decode the image (%s): %s", tt.pic, err)
			}
			rgba = img.(*image.NRGBA)
		}()

		var modelDir string
		var scaleFn string
		var noiseFn string

		switch tt.mode {
		case Anime:
			modelDir = "anime_style_art_rgb"
		case Photo:
			modelDir = "photo"
		}

		scaleFn = fmt.Sprintf("model/%s/scale2.0x_model.json", modelDir)
		if tt.noise > 0 {
			noiseFn = fmt.Sprintf("model/%s/noise%d_model.json", modelDir, tt.noise)
		}

		model2x, err := LoadModelAssets(scaleFn)
		if err != nil {
			b.Fatalf("failed to load scale2x model: %s", err)
		}

		var noise Model
		if tt.noise > 0 {
			noise, err = LoadModelAssets(noiseFn)
			if err != nil {
				b.Fatalf("failed to load noise model: %s", err)
			}
		}

		b.Run(tt.name, func(b *testing.B) {
			w2x := Waifu2x{
				scaleModel: model2x,
				noiseModel: noise,
				parallel:   runtime.NumCPU(),
			}

			b.ResetTimer()
			img := ChannelImage{
				Width:  rgba.Bounds().Max.X,
				Height: rgba.Bounds().Max.Y,
				Buffer: rgba.Pix,
			}
			if _, err := w2x.convertChannelImage(context.TODO(), img, tt.alpha, 2); err != nil {
				b.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAllCombinations(t *testing.T) {
	tests := []struct {
		name  string
		mode  Mode
		noise int
	}{
		{
			name:  "Anime, noiseModel reduction level 0",
			mode:  Anime,
			noise: 0,
		},
		{
			name:  "Anime, noiseModel reduction level 1",
			mode:  Anime,
			noise: 1,
		},
		{
			name:  "Anime, noiseModel reduction level 2",
			mode:  Anime,
			noise: 2,
		},
		{
			name:  "Anime, noiseModel reduction level 3",
			mode:  Anime,
			noise: 3,
		},
		{
			name:  "Photo, noiseModel reduction level 0",
			mode:  Photo,
			noise: 0,
		},
		{
			name:  "Photo, noiseModel reduction level 1",
			mode:  Photo,
			noise: 1,
		},
		{
			name:  "Photo, noiseModel reduction level 2",
			mode:  Photo,
			noise: 2,
		},
		{
			name:  "Photo, noiseModel reduction level 3",
			mode:  Photo,
			noise: 3,
		},
	}

	fn := "../testdata/neko_alpha.png"
	fp, err := os.Open(fn)
	if err != nil {
		t.Fatalf("failed to open %s: %s", fn, err)
	}
	defer fp.Close()

	img, err := png.Decode(fp)
	if err != nil {
		t.Fatalf("failed to decode the test pic (%s): %s", fn, err)
	}

	rgba := img.(*image.NRGBA)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var modelDir string
			var scaleFn string
			var noiseFn string

			switch tt.mode {
			case Anime:
				modelDir = "anime_style_art_rgb"
			case Photo:
				modelDir = "photo"
			}

			scaleFn = fmt.Sprintf("model/%s/scale2.0x_model.json", modelDir)
			if tt.noise > 0 {
				noiseFn = fmt.Sprintf("model/%s/noise%d_model.json", modelDir, tt.noise)
			}

			model2x, err := LoadModelAssets(scaleFn)
			if err != nil {
				t.Fatalf("failed to load scale2x model: %s", err)
			}

			var noise Model
			if tt.noise > 0 {
				noise, err = LoadModelAssets(noiseFn)
				if err != nil {
					t.Fatalf("failed to load noise model: %s", err)
				}
			}

			w2x := Waifu2x{
				scaleModel: model2x,
				noiseModel: noise,
				parallel:   runtime.NumCPU(),
			}
			img := ChannelImage{
				Buffer: rgba.Pix,
				Width:  rgba.Bounds().Max.X,
				Height: rgba.Bounds().Max.Y,
			}
			if _, err := w2x.convertChannelImage(context.TODO(), img, true, 2); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
