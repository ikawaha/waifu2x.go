package waifu2x

import (
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
			pic:   "testdata/neko_small.png",
			mode:  modeAnime,
			noise: 0,
			alpha: false,
		},
		{
			name:  "Wikipe-tan",
			pic:   "testdata/wikipetan.png",
			mode:  modeAnime,
			noise: 0,
			alpha: true,
		},
	}

	for _, tt := range tests {
		fp, err := os.Open(tt.pic)
		if err != nil {
			b.Fatalf("failed to open the image (%s): %s", tt.pic, err)
		}
		defer fp.Close()
		img, err := png.Decode(fp)
		if err != nil {
			b.Fatalf("failed to decode the image (%s): %s", tt.pic, err)
		}
		rgba := img.(*image.NRGBA)

		var modelDir string
		var scaleFn string
		var noiseFn string

		switch tt.mode {
		case modeAnime:
			modelDir = "anime_style_art_rgb"
		case modePhoto:
			modelDir = "photo"
		}

		scaleFn = fmt.Sprintf("models/%s/scale2.0x_model.json", modelDir)
		if tt.noise > 0 {
			noiseFn = fmt.Sprintf("models/%s/noise%d_model.json", modelDir, tt.noise)
		}

		model2x, err := LoadModelFromAssets(scaleFn)
		if err != nil {
			b.Fatalf("failed to load scale2x model: %s", err)
		}

		var noise *Model
		if tt.noise > 0 {
			noise, err = LoadModelFromAssets(noiseFn)
			if err != nil {
				b.Fatalf("failed to load noise model: %s", err)
			}
		}

		b.Run(tt.name, func(b *testing.B) {
			model := Waifu2x{
				Scale2xModel: model2x,
				NoiseModel:   noise,
				Scale:        2,
				Jobs:         runtime.NumCPU(),
			}

			b.ResetTimer()
			model.Calc(rgba.Pix, rgba.Bounds().Max.X, rgba.Bounds().Max.Y, tt.alpha)
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
			name:  "Anime, noise reduction level 0",
			mode:  modeAnime,
			noise: 0,
		},
		{
			name:  "Anime, noise reduction level 1",
			mode:  modeAnime,
			noise: 1,
		},
		{
			name:  "Anime, noise reduction level 2",
			mode:  modeAnime,
			noise: 2,
		},
		{
			name:  "Anime, noise reduction level 3",
			mode:  modeAnime,
			noise: 3,
		},
		{
			name:  "Photo, noise reduction level 0",
			mode:  modePhoto,
			noise: 0,
		},
		{
			name:  "Photo, noise reduction level 1",
			mode:  modePhoto,
			noise: 1,
		},
		{
			name:  "Photo, noise reduction level 2",
			mode:  modePhoto,
			noise: 2,
		},
		{
			name:  "Photo, noise reduction level 3",
			mode:  modePhoto,
			noise: 3,
		},
	}

	fn := "testdata/wikipetan.png"
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

			scaleFn = fmt.Sprintf("models/%s/scale2.0x_model.json", modelDir)
			if tt.noise > 0 {
				noiseFn = fmt.Sprintf("models/%s/noise%d_model.json", modelDir, tt.noise)
			}

			model2x, err := LoadModelFromAssets(scaleFn)
			if err != nil {
				t.Fatalf("failed to load scale2x model: %s", err)
			}

			var noise *Model
			if tt.noise > 0 {
				noise, err = LoadModelFromAssets(noiseFn)
				if err != nil {
					t.Fatalf("failed to load noise model: %s", err)
				}
			}

			model := Waifu2x{
				Scale2xModel: model2x,
				NoiseModel:   noise,
				Scale:        2,
				Jobs:         runtime.NumCPU(),
			}

			model.Calc(rgba.Pix, rgba.Bounds().Max.X, rgba.Bounds().Max.Y, true)
		})
	}
}
