package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/puhitaku/go-waifu2x/model"
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

	pix, enableAlphaUpscaling, err := model.ImageToPix(img)
	if err != nil {
		return fmt.Errorf("failed to extract pix from the image: %w", err)
	}

	ms, err := modelSet(opt.mode, opt.noiseReduction)
	if err != nil {
		return fmt.Errorf("failed to load models: %w", err)
	}

	w2x := model.Waifu2x{
		Scale2xModel: ms.Scale2xModel,
		NoiseModel:   ms.NoiseModel,
		Scale:        opt.scale,
		Jobs:         opt.jobs,
	}
	ci := model.ChannelImage{
		Width:  img.Bounds().Max.X,
		Height: img.Bounds().Max.Y,
		Buffer: pix,
	}
	ci = w2x.Calc(ci, enableAlphaUpscaling)

	var w io.Writer = os.Stdout
	if opt.output != "" {
		fp, err := os.Create(opt.output)
		if err != nil {
			return fmt.Errorf("output file, %w", err)
		}
		defer fp.Close()
		w = fp
	}
	rgba := ci.ToRGBA()
	if err := png.Encode(w, &rgba); err != nil {
		panic(err)
	}
	return nil
}

func modelSet(mode string, noiseLevel int) (*model.ModelSet, error) {
	switch mode {
	case modeAnime:
		return model.NewAnimeModelSet(noiseLevel)
	case modePhoto:
		return model.NewPhotoModelSet(noiseLevel)
	}
	return nil, fmt.Errorf("unknown model type: %s", mode)
}
