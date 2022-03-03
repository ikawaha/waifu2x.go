package cmd

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/gif"
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

func parseInputImage(file string) ([]byte, string, error) {
	r := os.Stdin
	if file != "" {
		var err error
		r, err = os.Open(file)
		if err != nil {
			return nil, "", err
		}
		defer r.Close()
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(b))
	return b, format, err
}

func decodeImage(b []byte, format string) (image.Image, error) {
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

func scaleUp(ctx context.Context, w2x *engine.Waifu2x, img image.Image, scale float64, w io.Writer) error {
	ci, err := w2x.ScaleUp(ctx, img, scale)
	if err != nil {
		return err
	}
	rgba := ci.ImageRGBA()
	if err := png.Encode(w, &rgba); err != nil {
		return fmt.Errorf("output error: %w", err)
	}
	return nil
}

func scaleUpGIF(ctx context.Context, w2x *engine.Waifu2x, img *gif.GIF, scale float64, w io.Writer) error {
	g, err := w2x.ScaleUpGIF(ctx, img, scale)
	if err != nil {
		return err
	}
	if err := gif.EncodeAll(w, g); err != nil {
		return fmt.Errorf("output error: %w", err)
	}
	return nil
}

// Run executes the waifu2x command.
func Run(args []string) error {
	opt := newOption(os.Stderr, flag.ExitOnError)
	if err := opt.parse(args); err != nil {
		return err
	}
	b, format, err := parseInputImage(opt.input)
	if err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	w2x, err := engine.NewWaifu2x(opt.mode, opt.noise, []engine.Option{
		engine.Verbose(opt.verbose),
		engine.Parallel(opt.parallel),
		engine.LogOutput(os.Stderr),
	}...)
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
	if format != "gif" {
		img, err := decodeImage(b, format)
		if err != nil {
			return err
		}
		return scaleUp(context.TODO(), w2x, img, opt.scale, w)
	}
	img, err := gif.DecodeAll(bytes.NewReader(b))
	if err != nil {
		return err
	}
	return scaleUpGIF(context.TODO(), w2x, img, opt.scale, w)
}
