package cmd

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/ikawaha/waifu2x.go/engine"
)

const (
	commandName  = "waifu2x"
	usageMessage = "%s (-i|--input) <input_file> [-o|--output <output_file>] [-s|--scale <scale_factor>] [-j|--jobs <n>] [-n|--noise <n>] [-m|--mode (anime|photo)]\n"
)

const (
	modeAnime = "anime"
	modePhoto = "photo"
)

type option struct {
	input          string
	output         string
	scale          float64
	jobs           int
	noiseReduction int
	mode           string
	flagSet        *flag.FlagSet
}

func newOption(w io.Writer, eh flag.ErrorHandling) (o *option) {
	o = &option{
		flagSet: flag.NewFlagSet(commandName, eh),
	}
	// option settings
	o.flagSet.SetOutput(w)
	o.flagSet.StringVar(&o.input, "i", "", "input file (short)")
	o.flagSet.StringVar(&o.input, "input", "", "input file")
	o.flagSet.StringVar(&o.output, "o", "", "output file (short) (default stdout)")
	o.flagSet.StringVar(&o.output, "output", "", "output file (default stdout)")
	o.flagSet.Float64Var(&o.scale, "s", 2.0, "scale multiplier >= 1.0 (short)")
	o.flagSet.Float64Var(&o.scale, "scale", 2.0, "scale multiplier >= 1.0")
	o.flagSet.IntVar(&o.jobs, "j", runtime.NumCPU(), "# of goroutines (short)")
	o.flagSet.IntVar(&o.jobs, "jobs", runtime.NumCPU(), "# of goroutines")
	o.flagSet.IntVar(&o.noiseReduction, "n", 0, "noise reduction level 0 <= n <= 3 (short)")
	o.flagSet.IntVar(&o.noiseReduction, "noise", 0, "noise reduction level 0 <= n <= 3")
	o.flagSet.StringVar(&o.mode, "m", modeAnime, "waifu2x mode, choose from 'anime' and 'photo' (short) (default anime)")
	o.flagSet.StringVar(&o.mode, "mode", modeAnime, "waifu2x mode, choose from 'anime' and 'photo' (default anime)")

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
	if o.input == "" {
		return fmt.Errorf("input file is empty")
	}
	if o.scale < 1.0 {
		return fmt.Errorf("invalid scale, %v > 1", o.scale)
	}
	if o.jobs < 1 {
		return fmt.Errorf("invalid number of jobs, %v < 1", o.jobs)
	}
	if o.noiseReduction < 0 || o.noiseReduction > 3 {
		return fmt.Errorf("invalid number of noise reduction level, it must be 0 - 3")
	}
	if o.mode != modeAnime && o.mode != modePhoto {
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

// Run executes the waifu2x command.
func Run(args []string) error {
	opt := newOption(os.Stderr, flag.ContinueOnError)
	if err := opt.parse(args); err != nil {
		return err
	}

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
