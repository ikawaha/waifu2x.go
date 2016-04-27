package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	commandName  = "waifu2x"
	usageMessage = "%s -input <input_file> [-output <output_file>] [-scale <scale_factor>]\n"
)

type option struct {
	scale   float64
	input   string
	output  string
	flagSet *flag.FlagSet
}

func newOption(w io.Writer, eh flag.ErrorHandling) (o *option) {
	o = &option{
		// ContinueOnError ErrorHandling // Return a descriptive error.
		// ExitOnError                   // Call os.Exit(2).
		// PanicOnError                  // Call panic with a descriptive error.flag.ContinueOnError
		flagSet: flag.NewFlagSet(commandName, eh),
	}
	// option settings
	o.flagSet.SetOutput(w)
	o.flagSet.Float64Var(&o.scale, "scale", 2.0, "scale >= 1.0")
	o.flagSet.StringVar(&o.input, "input", "", "input file")
	o.flagSet.StringVar(&o.output, "output", "", "output file (default stdout)")

	return
}

func (o *option) parse(args []string) (err error) {
	if err = o.flagSet.Parse(args); err != nil {
		return
	}
	// validations
	if nonFlag := o.flagSet.Args(); len(nonFlag) != 0 {
		return fmt.Errorf("invalid argument: %v", nonFlag)
	}
	if o.input == "" {
		return fmt.Errorf("input file is empty\n")
	}
	if o.scale < 1.0 {
		return fmt.Errorf("invalid scale, %v > 1", o.scale)
	}
	return
}

func Usage() {
	fmt.Printf(usageMessage, commandName)
}

func PrintDefaults(eh flag.ErrorHandling) {
	o := newOption(os.Stderr, eh)
	o.flagSet.PrintDefaults()
}

func main() {
	opt := newOption(os.Stderr, flag.ExitOnError)
	if err := opt.parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		Usage()
		PrintDefaults(flag.ExitOnError)
		os.Exit(1)
	}
	if err := run(opt); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
