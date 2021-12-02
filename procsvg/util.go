package main

import (
	"io"
	"os"
	"path/filepath"
)

func readdirnames(dir, pattern string) ([]string, error) {
	// check for bat pattern
	_, err := filepath.Match(pattern, "")
	if err != nil {
		return nil, err
	}

	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var matches []string

	for {
		names, err := f.Readdirnames(100)

		for _, n := range names {
			if m, _ := filepath.Match(pattern, n); m {
				matches = append(matches, n)
			}
		}

		if err == io.EOF {
			return matches, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func file_up_to_date(src, dest string) bool {
	si, err := os.Stat(src)
	if err != nil {
		return false
	}

	di, err := os.Stat(dest)
	if err != nil {
		return false
	}

	return si.ModTime().Before(di.ModTime())
}
