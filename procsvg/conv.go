package main

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	"os"
	"strconv"
	"strings"
)

func ConvertSvg(fn string) (*ProgImage, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	g := svgprog{fn: fn}

	d := xml.NewDecoder(f)
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if x, ok := tok.(xml.StartElement); ok {
			if err := g.elem(x); err != nil {
				return nil, err
			}
		}
	}

	return g.finish(), nil
}

type svgprog struct {
	fn  string
	im  *ProgImage
	mem ProgMem
}

func (g *svgprog) finish() *ProgImage {
	g.mem.Stop()
	g.im.Data = g.mem.Bytes()
	return g.im
}

func (g *svgprog) elem(e xml.StartElement) error {
	if hasattr(e, "clip-path") {
		fmt.Fprintln(os.Stderr, "clip-path found in %s", g.fn)
	}
	if hasattr(e, "transform") {
		fmt.Fprintln(os.Stderr, "transform found in %s", g.fn)
	}

	switch e.Name.Local {
	case "svg":
		return g.svg(e)

	case "path":
		return g.path(e)
	}

	return nil
}

func (g *svgprog) svg(e xml.StartElement) error {
	g.im = new(ProgImage)

	var err error

	ws := findattr(e, "width")
	g.im.Width, err = strconv.Atoi(strings.TrimSuffix(ws, "px"))
	if err != nil {
		return fmt.Errorf("error parsing width %q", ws)
	}

	hs := findattr(e, "height")
	g.im.Height, err = strconv.Atoi(strings.TrimSuffix(hs, "px"))
	if err != nil {
		return fmt.Errorf("error parsing width %q", hs)
	}

	vb := findattr(e, "viewBox")
	var minx, miny, width, height float64
	_, err = fmt.Sscan(vb, &minx, &miny, &width, &height)
	if err != nil {
		return fmt.Errorf("error parsing viewBox %q", vb)
	}

	g.mem.ViewBox(minx, miny, minx+width, miny+height)
	return nil
}

func (g *svgprog) path(e xml.StartElement) error {
	if err := g.handle_fill(e); err != nil {
		return err
	}

	cmds, err := PathDCmds(findattr(e, "d"))
	if err != nil {
		return err
	}

	for _, c := range cmds {
		if err := g.mem.PathCmd(c); err != nil {
			return err
		}
	}

	return nil
}

func (g *svgprog) handle_fill(e xml.StartElement) error {
	c, ok := get_svg_solid_fill(e)
	if !ok {
		return fmt.Errorf("can't find fill style")
	}

	g.mem.Byte(0x01)
	g.mem.Color(c)

	return nil
}

func hasattr(e xml.StartElement, name string) bool {
	for _, a := range e.Attr {
		if a.Name.Local == name {
			return true
		}
	}
	return false
}

func findattr(e xml.StartElement, name string) string {
	var r string
	found := false
	for _, a := range e.Attr {
		if a.Name.Local == name {
			if found {
				panic(fmt.Errorf("duplicate attr %s", name))
			}
			r = a.Value
			found = true
		}
	}
	return r
}

func get_svg_solid_fill(e xml.StartElement) (color.Color, bool) {
	fs := findattr(e, "fill")
	if fs != "" {
		c, ok := colorfromhex(fs)
		if ok {
			return c, true
		}
	}

	style := cssdecode(findattr(e, "style"))
	return colorfromhex(style["fill"])
}

func colorfromhex(s string) (color.Color, bool) {
	if s == "" || s[0] != '#' {
		return nil, false
	}

	v, err := strconv.ParseUint(s[1:], 16, 64)
	if err != nil {
		return nil, false
	}

	switch len(s) - 1 {
	case 3:
		r := (byte(v>>8) & 0xf) * 0x11
		g := (byte(v>>4) & 0xf) * 0x11
		b := (byte(v) & 0xf) * 0x11
		return color.NRGBA{r, g, b, 0xff}, true

	case 6:
		r := byte(v >> 16)
		g := byte(v >> 8)
		b := byte(v)
		return color.NRGBA{r, g, b, 0xff}, true
	}

	return nil, false
}

func cssdecode(css string) map[string]string {
	if css == "" {
		return nil
	}

	v := strings.Split(css, ";")
	m := make(map[string]string)
	for _, e := range v {
		colon := strings.Index(e, ":")
		if colon > 0 {
			m[strings.TrimSpace(e[:colon])] = strings.TrimSpace(e[colon+1:])
		}
	}
	return m
}
