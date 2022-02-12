package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/puhitaku/go-waifu2x"
)

func run(opt *option) error {
	fp, err := os.Open(opt.input)
	if err != nil {
		return fmt.Errorf("input file %v, %w", opt.input, err)
	}
	defer fp.Close()

	var img image.Image
	if strings.HasSuffix(fp.Name(), "jpg") || strings.HasSuffix(fp.Name(), "jpeg") {
		img, err = jpeg.Decode(fp)
		if err != nil {
			return fmt.Errorf("load file %v, %w", opt.input, err)
		}
	} else if strings.HasSuffix(fp.Name(), "png") {
		img, err = png.Decode(fp)
		if err != nil {
			return fmt.Errorf("load file %v, %w", opt.input, err)
		}
	}

	pix, enableAlphaUpscaling, err := waifu2x.ImageToPix(img)
	if err != nil {
		return fmt.Errorf("failed to extract pix from the image: %w", err)
	}

	var modelDir string
	var scaleFn string
	var noiseFn string

	switch opt.mode {
	case modeAnime:
		modelDir = "anime_style_art_rgb"
	case modePhoto:
		modelDir = "photo"
	}

	scaleFn = fmt.Sprintf("models/%s/scale2.0x_model.json", modelDir)
	if opt.noiseReduction > 0 {
		noiseFn = fmt.Sprintf("models/%s/noise%d_model.json", modelDir, opt.noiseReduction)
	}

	model2x, err := waifu2x.LoadModelFromAssets(scaleFn)
	if err != nil {
		return fmt.Errorf("failed to load scale2x model: %w", err)
	}

	var noise *waifu2x.Model
	if opt.noiseReduction > 0 {
		noise, err = waifu2x.LoadModelFromAssets(noiseFn)
		if err != nil {
			return fmt.Errorf("failed to load noise model: %w", err)
		}
	}

	model := waifu2x.Waifu2x{
		Scale2xModel: model2x,
		NoiseModel:   noise,
		Scale:        opt.scale,
		Jobs:         opt.jobs,
	}

	pix, rect := model.Calc(pix, img.Bounds().Max.X, img.Bounds().Max.Y, enableAlphaUpscaling)

	var w io.Writer = os.Stdout
	if opt.output != "" {
		fp, err := os.Create(opt.output)
		if err != nil {
			return fmt.Errorf("output file, %w", err)
		}
		defer fp.Close()
		w = fp
	}
	if err := png.Encode(w, waifu2x.PixToRGBA(pix, rect)); err != nil {
		panic(err)
	}
	return nil
}
