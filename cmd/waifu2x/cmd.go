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

	ms, err := modelSet(opt.mode, opt.noiseReduction)
	if err != nil {
		return fmt.Errorf("failed to load models: %w", err)
	}

	model := waifu2x.Waifu2x{
		Scale2xModel: ms.Scale2xModel,
		NoiseModel:   ms.NoiseModel,
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

func modelSet(mode string, noiseLevel int) (*waifu2x.ModelSet, error) {
	switch mode {
	case modeAnime:
		return waifu2x.NewAnimeModelSet(noiseLevel)
	case modePhoto:
		return waifu2x.NewPhotoModelSet(noiseLevel)
	}
	return nil, fmt.Errorf("unknown model type: %s", mode)
}
