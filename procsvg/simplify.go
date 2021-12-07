package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func simplify_svg(project Project) error {

	inkscape := NewInkscapeShell()
	defer inkscape.Close()

	for _, sub := range project.SizeDir {
		sd := filepath.Join(project.IconDir, sub)
		td := filepath.Join(project.IntermediateDir, sub)

		if err := simplify_svg_dir(inkscape, sd, td); err != nil {
			return err
		}
	}

	return nil
}

func simplify_svg_dir(is *InkscapeShell, sd, td string) error {
	if err := os.MkdirAll(td, 0666); err != nil {
		return err
	}

	svgs, err := readdirnames(sd, "*.svg")
	if err != nil {
		return err
	}

	for _, svg := range svgs {
		sf := filepath.Join(sd, svg)
		tf := filepath.Join(td, svg)
		if err := simplify_svg_file(is, sf, tf); err != nil {
			return err
		}
	}

	return nil
}

func simplify_svg_file(is *InkscapeShell, sf, tf string) error {
	if !cli.rebuild && file_up_to_date(sf, tf) {
		return nil
	}

	if cli.verbose {
		fmt.Printf("Simplify %s\n", sf)
	}

	is.Cmdf("file-open:%s", filepath.ToSlash(sf))
	is.Cmd("select-all")
	is.Cmd("object-to-path")
	is.Cmd("object-stroke-to-path")
	is.Cmd("export-overwrite:true")
	is.Cmd("export-plain-svg:true")
	is.Cmdf("export-filename:%s", filepath.ToSlash(tf))
	is.Cmd("export-do")
	is.Cmd("file-close")

	return is.Err()
}
