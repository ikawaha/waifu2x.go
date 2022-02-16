package main

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/puhitaku/go-waifu2x/engine"
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

	mode := engine.Anime
	switch opt.mode {
	case "anime":
		mode = engine.Anime
	case "photo":
		mode = engine.Photo
	}
	w2x, err := engine.NewWaifu2x(mode, opt.noiseReduction, engine.Parallel(8), engine.Verbose())
	if err != nil {
		return err
	}

	rgba, err := w2x.ScaleUp(context.TODO(), img, opt.scale)
	if err != nil {
		return fmt.Errorf("calc error: %w", err)
	}

	var w io.Writer = os.Stdout
	if opt.output != "" {
		fp, err := os.Create(opt.output)
		if err != nil {
			return fmt.Errorf("output file, %w", err)
		}
		defer fp.Close()
		w = fp
	}
	if err := png.Encode(w, &rgba); err != nil {
		panic(err)
	}
	return nil
}
