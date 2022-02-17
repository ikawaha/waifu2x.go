package main

import (
	"fmt"
	"os"

	"github.com/puhitaku/go-waifu2x/cmd"
)

func main() {
	if err := cmd.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		cmd.Usage()
		os.Exit(1)
	}
	os.Exit(0)
}
