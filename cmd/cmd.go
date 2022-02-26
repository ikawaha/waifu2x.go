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
	"runtime"

	"github.com/ikawaha/waifu2x.go/engine"
)

const (
	modeAnime = "anime"
	modePhoto = "photo"
)

type option struct {
	// flagSet args
	input    string
	output   string
	scale    float64
	noise    int
	parallel int
	modeStr  string
	verbose  bool

	// option values
	mode    engine.Mode
	flagSet *flag.FlagSet
}

const commandName = `waifu2x`

func newOption(w io.Writer, eh flag.ErrorHandling) (o *option) {
	o = &option{
		flagSet: flag.NewFlagSet(commandName, eh),
	}
	// option settings
	o.flagSet.SetOutput(w)
	o.flagSet.StringVar(&o.input, "i", "", "input file (default stdin)")
	o.flagSet.StringVar(&o.output, "o", "", "output file (default stdout)")
	o.flagSet.Float64Var(&o.scale, "s", 2.0, "scale multiplier >= 1.0")
	o.flagSet.IntVar(&o.noise, "n", 0, "noise reduction level 0 <= n <= 3")
	o.flagSet.IntVar(&o.parallel, "p", runtime.GOMAXPROCS(runtime.NumCPU()), "concurrency")
	o.flagSet.StringVar(&o.modeStr, "m", modeAnime, "waifu2x mode, choose from 'anime' and 'photo'")
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
	if o.noise < 0 || o.noise > 3 {
		return fmt.Errorf("invalid number of noise reduction level, it must be [0,3]")
	}
	if o.parallel < 1 {
		return fmt.Errorf("invalid number of parallel, it must be >= 1")
	}
	switch o.modeStr {
	case modeAnime:
		o.mode = engine.Anime
	case modePhoto:
		o.mode = engine.Photo
	default:
		return fmt.Errorf("invalid mode, choose from 'anime' or 'photo'")
	}
	return nil
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
	opt := newOption(os.Stderr, flag.ExitOnError)
	if err := opt.parse(args); err != nil {
		return err
	}
	img, err := parseInputImage(opt.input)
	if err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	w2x, err := engine.NewWaifu2x(opt.mode, opt.noise, []engine.Option{
		engine.Verbose(opt.verbose),
		engine.Parallel(opt.parallel),
	}...)
	if err != nil {
		return err
	}
	rgba, err := w2x.ScaleUp(context.TODO(), img, opt.scale)
	if err != nil {
		return err
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
