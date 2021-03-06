package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var cli struct {
	rebuild   bool
	verbose   bool
	showColor bool
	disasm    bool

	inkscape string
}

func main() {
	flag.BoolVar(&cli.rebuild, "r", false, "rebuild intermediate icons")
	flag.BoolVar(&cli.verbose, "v", false, "verbose operation")
	flag.BoolVar(&cli.showColor, "showcolor", false, "show icon colors")
	flag.BoolVar(&cli.disasm, "disasm", false, "write disassembly")
	flag.StringVar(&cli.inkscape, "inkscape", "", "inkscape path (default: $PROCSVG_INKSCAPE or $PATH)")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "No project files")
		os.Exit(1)
	}

	for _, arg := range flag.Args() {
		if err := run_project(arg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func run_project(fn string) error {
	project, err := LoadProject(fn)
	if err != nil {
		return fmt.Errorf("Error loading project file %s: %w", fn, err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Error getting current directory: %w", err)
	}

	pd := filepath.Dir(fn)
	if err := os.Chdir(pd); err != nil {
		return fmt.Errorf("Error changing directory: %w", err)
	}
	defer os.Chdir(wd)

	if err := simplify_svg(project); err != nil {
		return err
	}

	return pack_project(project)
}
