package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	fmt.Fprintln(os.Stderr, "structviz — Go struct memory layout visualizer")
	fmt.Fprintln(os.Stderr, "Paste your struct definition below, then press Ctrl+D (Unix/Mac) or Ctrl+Z (Windows):")
	fmt.Fprintln(os.Stderr)

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fatalf("error reading input: %v", err)
	}

	input := strings.TrimSpace(string(data))
	if input == "" {
		fatalf("no input provided")
	}

	fields, name, err := parseStruct(input)
	if err != nil {
		fatalf("parse error: %v", err)
	}

	current := computeLayout(fields)
	optimized := optimizeLayout(fields)

	// Original layout
	fmt.Print(banner(fmt.Sprintf("ORIGINAL: %s", name)))
	fmt.Print(statsLine(current))
	fmt.Println()
	fmt.Print(renderGrid(current))
	fmt.Println()
	fmt.Print(renderLegend(current))

	// Optimized layout
	fmt.Print(banner(fmt.Sprintf("OPTIMIZED: %s", name)))
	fmt.Print(statsLine(optimized))

	savedBytes := current.StructSize - optimized.StructSize
	if savedBytes > 0 {
		fmt.Printf("  Saved %d bytes (%.0f%% reduction)\n", savedBytes,
			float64(savedBytes)/float64(current.StructSize)*100)
	} else {
		fmt.Println("  Already optimal — no savings possible.")
	}

	fmt.Println()
	fmt.Println(formatStruct(name, optimized))
	fmt.Println()
	fmt.Print(renderGrid(optimized))
	fmt.Println()
	fmt.Print(renderLegend(optimized))
	fmt.Println()
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
