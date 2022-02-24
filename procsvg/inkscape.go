package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type InkscapeShell struct {
	started bool
	err     error

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
}

func NewInkscapeShell() *InkscapeShell {
	ib, err := inkscapebin_checked()
	if err != nil {
		return &InkscapeShell{err: err}
	}

	c := exec.Command(ib, "--shell")

	c.Stderr = os.Stderr

	si, err := c.StdinPipe()
	if err != nil {
		return &InkscapeShell{err: fmt.Errorf("Error setting up stdin pipe: %w", err)}
	}

	so, err := c.StdoutPipe()
	if err != nil {
		return &InkscapeShell{err: fmt.Errorf("Error setting up stdout pipe: %w", err)}
	}

	return &InkscapeShell{cmd: c, stdin: si, stdout: bufio.NewScanner(so)}
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

	if !strings.HasSuffix(cmd, "\n") {
		cmd += "\n"
	}
	if _, err := fmt.Fprintln(is.stdin, cmd); err != nil {
		is.err = err
		return
	}

	cmd = strings.TrimRight(cmd, "\n")
	if i := strings.LastIndexByte(cmd, '\n'); i > 0 {
		cmd = cmd[i+1:]
	}
	if err := is.sync("> " + cmd); err != nil {
		is.err = err
	}
}

func (is *InkscapeShell) Cmdf(format string, args ...interface{}) {
	if !is.start() {
		return
	}

	is.Cmd(fmt.Sprintf(format, args...))
}

func (is *InkscapeShell) Close() error {
	var err error
	if is.started {
		is.Cmd("quit")
		is.stdin.Close()
		err = is.sync("")
		if xerr := is.cmd.Wait(); err == nil {
			err = xerr
		}
	}

	if is.err != nil {
		return is.err
	}

	return err
}

func (is *InkscapeShell) sync(wantLine string) error {
	scanner := is.stdout

	for scanner.Scan() {
		t := scanner.Text()
		if cli.verbose && t != "> " {
			fmt.Println(t)
		}
		if wantLine != "" && t == wantLine {
			return nil
		}
	}

	return scanner.Err()
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

	sout := string(out)
	for _, line := range strings.SplitAfter(sout, "\n") {
		if strings.HasPrefix(line, "Inkscape") {
			if line < "Inkscape 1.2" {
				return "", fmt.Errorf("Needs inkscape version 1.2+, got %v", line)
			} else {
				return v, nil
			}
		}
	}

	return "", fmt.Errorf("Can't recognize inkscape version: %v", sout)
}

func inkscapebin() (string, error) {
	if cli.inkscape != "" {
		return cli.inkscape, nil
	}

	if v := os.Getenv("PROCSVG_INKSCAPE"); v != "" {
		return v, nil
	}

	return exec.LookPath("inkspace")
}
