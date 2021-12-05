package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

type pathSvgWriter struct {
	w io.WriteCloser
	n int
}

func newPathSvgWriter(t *testing.T) *pathSvgWriter {
	const (
		x, y   = -100, -100
		dx, dy = 200, 200
	)

	f, err := os.Create("testpath.svg")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Fprintf(f, `<?xml version="1.0" encoding="UTF-8"?>
<svg version="1.1"
 xmlns="http://www.w3.org/2000/svg"
 xmlns:xlink="http://www.w3.org/1999/xlink"
 x="%dpx" y="%dpx" width="%dpx" height="%dpx"
 viewBox="%d %d %d %d"
 xml:space="preserve">
`, x, y, dx, dy, x, y, dx, dy)

	return &pathSvgWriter{w: f}
}

func (w *pathSvgWriter) Close() error {
	fmt.Fprintln(w.w, "</svg>")
	return w.w.Close()
}

func (w *pathSvgWriter) arct(z testArc) {
	w.n++
	fmt.Fprintf(w.w, ` <g id="layer%03d">`+"\n", w.n)

	arcno := 800 + 2*w.n
	curveno := arcno + 1

	arc := fmt.Sprintf("M%f,%fA%f,%f %f %d %d %f,%fz",
		z.p1x, z.p1y, z.rx, z.ry, z.angle,
		boolnum(z.largeArc), boolnum(z.sweep), z.p2x, z.p2y)
	fmt.Fprintf(w.w, `  <path id="path%d" d="%s" fill="none" stroke="black" stroke-width="0.1"/>`,
		arcno, arc)

	pts := arcToBezier(Point{z.p1x, z.p1y}, Point{z.p2x, z.p2y},
		Point{z.rx, z.ry}, z.angle, z.largeArc, z.sweep)

	if len(pts) != 0 {
		sb := &strings.Builder{}
		fmt.Fprintf(sb, "M%f,%f C", z.p1x, z.p1y)
		for _, p := range pts {
			fmt.Fprintf(sb, " %f,%f", p.X, p.Y)
		}
		sb.WriteString("z")
		fmt.Fprintf(w.w, `  <path id="path%d" d="%s" fill="none" stroke="red" stroke-width="0.1"/>`,
			curveno, sb.String())
	}

	fmt.Fprintln(w.w, ` </g>`)
}

func boolnum(f bool) int {
	if f {
		return 1
	}
	return 0
}

type testArc struct {
	p1x, p1y float64
	rx, ry   float64
	angle    float64
	largeArc bool
	sweep    bool
	p2x, p2y float64
}

func TestArc2Bézier(t *testing.T) {

	tests := []testArc{
		{0, 20, 50, 20, 0, false, false, -50, 0},
		{0, 20, 50, 20, 0, true, false, -50, 0},
		{0, 20, 50, 20, 0, true, true, -50, 0},
		{0, 20, 50, 20, 0, false, true, -50, 0},
		{0, 20, 50, 20, 10, false, false, -50, 0},
		{0, 20, 50, 20, 20, false, false, -50, 0},
		{0, 20, 50, 20, 30, false, false, -50, 0},

		// d="M 29.4,15.5 A 13.9,13.9 0 0 1 15.5,29.4 13.9,13.9 0 0 1 1.6000004,15.5 13.9,13.9 0 0 1 15.5,1.6000004 13.9,13.9 0 0 1 29.4,15.5 Z"
		{29.4, 15.5, 13.9, 13.9, 0, false, true, 15.5, 29.4},
		{15.5, 29.4, 13.9, 13.9, 0, false, true, 1.6, 15.5},
		{1.6, 15.5, 13.9, 13.9, 0, false, true, 15.5, 1.6},
		{15.5, 1.6, 13.9, 13.9, 0, false, true, 29.4, 15.5},
	}

	w := newPathSvgWriter(t)
	defer w.Close()

	for _, tt := range tests {
		w.arct(tt)
	}
}
