package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
)

const (
	commandName  = "waifu2x"
	usageMessage = "%s -i|--input <input_file> [-o|--output <output_file>] [-s|--scale <scale_factor>] [-j|--jobs <n>] [-n|--noise <n>] [-m|--mode (anime|photo)]\n"
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

func usage() {
	fmt.Printf(usageMessage, commandName)
}

func main() {
	opt := newOption(os.Stderr, flag.ContinueOnError)
	if err := opt.parse(os.Args[1:]); err != nil {
		usage()
		opt.flagSet.PrintDefaults()
		os.Exit(2)
	}
	if err := run(opt); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
