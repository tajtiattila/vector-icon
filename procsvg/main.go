package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var cli struct {
	rebuild bool
	verbose bool
}

func main() {
	flag.BoolVar(&cli.rebuild, "r", false, "rebuild all icons")
	flag.BoolVar(&cli.verbose, "v", false, "verbose operation")
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
	project := DefaultProject
	if err := loadjson(fn, &project); err != nil {
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

	return simplify_svg(project)
}

func loadjson(fn string, v interface{}) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	return d.Decode(v)
}
