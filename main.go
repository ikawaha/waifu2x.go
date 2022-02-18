package main

import (
	"fmt"
	"os"

	"github.com/ikawaha/waifu2x.go/cmd"
)

func main() {
	if err := cmd.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		cmd.Usage()
		os.Exit(1)
	}
	os.Exit(0)
}
