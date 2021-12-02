package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type InkscapeShell struct {
	started bool
	err     error

	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func NewInkscapeShell() *InkscapeShell {
	ib, err := inkscapebin_checked()
	if err != nil {
		return &InkscapeShell{err: err}
	}

	c := exec.Command(ib, "--shell")

	c.Stderr = os.Stderr
	if cli.verbose {
		c.Stdout = os.Stdout
	}

	w, err := c.StdinPipe()
	if err != nil {
		return &InkscapeShell{err: fmt.Errorf("Error setting up stdin pipe: %w", err)}
	}

	return &InkscapeShell{cmd: c, stdin: w}
}

func (is *InkscapeShell) Err() error {
	return is.err
}

func (is *InkscapeShell) start() bool {
	if is.err != nil {
		return false
	}

	if is.started {
		return true
	}

	if err := is.cmd.Start(); err != nil {
		is.err = fmt.Errorf("Error starting inkscape: %w", err)
		return false
	}

	is.started = true
	return true
}

func (is *InkscapeShell) Cmd(cmd string) {
	if !is.start() {
		return
	}

	fmt.Fprintln(is.stdin, cmd)
}

func (is *InkscapeShell) Cmdf(format string, args ...interface{}) {
	if !is.start() {
		return
	}

	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(is.stdin, format, args...)
}

func (is *InkscapeShell) Close() error {
	var err error
	if is.started {
		is.Cmd("quit")
		is.stdin.Close()
		err = is.cmd.Wait()
	}

	if is.err != nil {
		return is.err
	}

	return err
}

func inkscapebin_checked() (string, error) {
	v, err := inkscapebin()
	if err != nil {
		return "", fmt.Errorf("Cannot find inkscape: %w", err)
	}

	out, err := exec.Command(v, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Error getting inkscape version: %w", err)
	}

	i := bytes.IndexByte(out, '\n')
	if i < 0 {
		i = len(out)
	}
	line := string(out[:i])
	if line < "Inkscape 1.2" {
		return "", fmt.Errorf("Needs inkscape version 1.2+, got %v", line)
	}

	return v, nil
}

func inkscapebin() (string, error) {
	if v := os.Getenv("PROCSVG_INKSCAPE"); v != "" {
		return v, nil
	}

	return exec.LookPath("inkspace")
}
