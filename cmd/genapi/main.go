// Command genapi reads internal/spec/api.json and emits api/*.gen.go.
//
// Usage:
//
//	genapi -input <file>     (default: internal/spec/api.json)
//	genapi -outdir <dir>     (default: api)
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	input := flag.String("input", "internal/spec/api.json", "IR JSON path")
	outdir := flag.String("outdir", "api", "output directory")
	flag.Parse()

	if err := run(*input, *outdir); err != nil {
		fmt.Fprintln(os.Stderr, "genapi:", err)
		os.Exit(1)
	}
}

// run is filled in by P2.T8/T9/T10.
func run(input, outdir string) error {
	api, err := loadAPI(input)
	if err != nil {
		return fmt.Errorf("load api: %w", err)
	}
	if err := os.MkdirAll(outdir, 0o750); err != nil {
		return err
	}
	e := newEmitter(api, outdir)
	if err := e.emitTypes(); err != nil {
		return err
	}
	if err := e.emitMethods(); err != nil {
		return err
	}
	if err := e.emitEnums(); err != nil {
		return err
	}
	if err := e.emitTests(); err != nil {
		return err
	}
	return nil
}
