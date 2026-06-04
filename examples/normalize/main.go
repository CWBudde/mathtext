// normalize reads display text from arguments (or stdin if none are given)
// and prints the Unicode fallback rendering. This shows the no-layout path
// for callers that only need plain-text substitution.
//
// Usage:
//
//	go run ./examples/normalize 'signal $\alpha_i^2$ peak'
//	echo 'energy is $E=mc^2$' | go run ./examples/normalize
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/cwbudde/mathtext"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [TEXT...]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "  With no arguments, reads lines from stdin.")
	}
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Println(mathtext.NormalizeDisplay(strings.Join(flag.Args(), " ")))
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		fmt.Println(mathtext.NormalizeDisplay(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "read error:", err)
		os.Exit(1)
	}
}
