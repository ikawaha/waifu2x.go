package cmd

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"

	"github.com/ikawaha/waifu2x.go/engine"
)

const (
	commandName  = "waifu2x"
	usageMessage = "%s [-i <input_file>] [-o <output_file>] [-s <scale_factor>] [-n <noise_reduction_level>] [-m (anime|photo)]\n"
)

const (
	modeAnime = "anime"
	modePhoto = "photo"
)

type option struct {
	// flagSet args
	input          string
	output         string
	scale          float64
	noiseReduction int
	modeStr        string
	verbose        bool
	// option values
	mode    engine.Mode
	flagSet *flag.FlagSet
}

func newOption(w io.Writer, eh flag.ErrorHandling) (o *option) {
	o = &option{
		flagSet: flag.NewFlagSet(commandName, eh),
	}
	// option settings
	o.flagSet.SetOutput(w)
	o.flagSet.StringVar(&o.input, "i", "", "input file (default stdin)")
	o.flagSet.StringVar(&o.output, "o", "", "output file (default stdout)")
	o.flagSet.Float64Var(&o.scale, "s", 2.0, "scale multiplier >= 1.0 (default 2)")
	o.flagSet.IntVar(&o.noiseReduction, "n", 0, "noise reduction level 0 <= n <= 3 (default 0)")
	o.flagSet.StringVar(&o.modeStr, "m", modeAnime, "waifu2x mode, choose from 'anime' and 'photo' (default anime)")
	o.flagSet.BoolVar(&o.verbose, "v", false, "verbose")
	return
}

func (o *option) parse(args []string) error {
	if err := o.flagSet.Parse(args); err != nil {
		return err
	}
	// validations
	if nonFlag := o.flagSet.Args(); len(nonFlag) != 0 {
		return fmt.Errorf("invalid argument: %v", nonFlag)
	}
	if o.scale < 1.0 {
		return fmt.Errorf("invalid scale, %v > 1", o.scale)
	}
	if o.noiseReduction < 0 || o.noiseReduction > 3 {
		return fmt.Errorf("invalid number of noise reduction level, it must be [0,3]")
	}
	switch o.modeStr {
	case "":
		o.mode = engine.Anime // default mode
	case modeAnime:
		o.mode = engine.Anime
	case modePhoto:
		o.mode = engine.Photo
	default:
		return fmt.Errorf("invalid mode, choose from 'anime' or 'photo'")
	}
	return nil
}

// Usage shows a usage message.
func Usage() {
	fmt.Printf(usageMessage, commandName)
	opt := newOption(os.Stdout, flag.ContinueOnError)
	opt.flagSet.PrintDefaults()
}

func parseInputImage(file string) (image.Image, error) {
	var b []byte
	in := os.Stdin
	if file != "" {
		var err error
		b, err = os.ReadFile(file)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		b, err = io.ReadAll(in)
		if err != nil {
			return nil, err
		}
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	var decoder func(io.Reader) (image.Image, error)
	switch format {
	case "jpeg":
		decoder = jpeg.Decode
	case "png":
		decoder = png.Decode
	default:
		return nil, fmt.Errorf("unsupported image type: %s", format)
	}
	return decoder(bytes.NewReader(b))
}

// Run executes the waifu2x command.
func Run(args []string) error {
	opt := newOption(os.Stderr, flag.ContinueOnError)
	if err := opt.parse(args); err != nil {
		return err
	}
	img, err := parseInputImage(opt.input)
	if err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	w2x, err := engine.NewWaifu2x(opt.mode, opt.noiseReduction, engine.Verbose(opt.verbose))
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
		return fmt.Errorf("output error: %w", err)
	}
	return nil
}
