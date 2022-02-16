package engine

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"runtime"
	"testing"
)

const (
	modeAnime = "anime"
	modePhoto = "photo"
)

func BenchmarkWaifu(b *testing.B) {
	tests := []struct {
		name  string
		pic   string
		mode  string
		noise int
		alpha bool
	}{
		{
			name:  "Neko",
			pic:   "../testdata/neko_small.png",
			mode:  modeAnime,
			noise: 0,
			alpha: false,
		},
		{
			name:  "Neko-alpha",
			pic:   "../testdata/neko_alpha.png",
			mode:  modeAnime,
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
		case modeAnime:
			modelDir = "anime_style_art_rgb"
		case modePhoto:
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
			w2x.convertChannelImage(context.TODO(), img, tt.alpha, 2)
		})
	}
}

func TestAllCombinations(t *testing.T) {
	tests := []struct {
		name  string
		mode  string
		noise int
	}{
		{
			name:  "Anime, noiseModel reduction level 0",
			mode:  modeAnime,
			noise: 0,
		},
		{
			name:  "Anime, noiseModel reduction level 1",
			mode:  modeAnime,
			noise: 1,
		},
		{
			name:  "Anime, noiseModel reduction level 2",
			mode:  modeAnime,
			noise: 2,
		},
		{
			name:  "Anime, noiseModel reduction level 3",
			mode:  modeAnime,
			noise: 3,
		},
		{
			name:  "Photo, noiseModel reduction level 0",
			mode:  modePhoto,
			noise: 0,
		},
		{
			name:  "Photo, noiseModel reduction level 1",
			mode:  modePhoto,
			noise: 1,
		},
		{
			name:  "Photo, noiseModel reduction level 2",
			mode:  modePhoto,
			noise: 2,
		},
		{
			name:  "Photo, noiseModel reduction level 3",
			mode:  modePhoto,
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
			case modeAnime:
				modelDir = "anime_style_art_rgb"
			case modePhoto:
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
			w2x.convertChannelImage(context.TODO(), img, true, 2)
		})
	}
}
