package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func PathDCmds(pathd string) ([]PathCmd, error) {
	d := pathdecoder{data: pathd}
	d.run()

	if d.err != nil {
		fmt.Println(pathd)
		fmt.Println(d.cmd)
		return nil, d.err
	}

	return d.cmd, nil
}

type Point struct {
	X, Y float64
}

type PathCmd struct {
	// path command, one of 'M', 'L', 'C' or 'Q'
	Cmd byte

	// command coords (always absolute)
	Pt []Point
}

func (c PathCmd) String() string {
	sb := new(strings.Builder)
	sb.WriteByte('{')
	sb.WriteByte(c.Cmd)
	for _, pt := range c.Pt {
		fmt.Fprintf(sb, " %v,%v", pt.X, pt.Y)
	}
	sb.WriteByte('}')
	return sb.String()
}

type pathdecoder struct {
	data string
	pos  int

	err error

	first Point
	last  Point
	lastc Point
	lastq Point

	cmd []PathCmd
}

func (d *pathdecoder) run() {
	for d.err == nil && d.pos < len(d.data) {
		d.step()
	}
}

func (d *pathdecoder) step() {
	c, relflag := d.nextcmd()

	rel := func(p Point) Point {
		if relflag {
			return Point{
				X: d.last.X + p.X,
				Y: d.last.Y + p.Y,
			}
		} else {
			return p
		}
	}

	switch c {

	case 'M':
		pts := d.points()
		p0 := rel(pts[0])
		d.addcmd('M', p0)
		d.line(pts[1:], rel)

	case 'L':
		d.line(d.points(), rel)

	case 'H':
		// horizontal lines
		for _, n := range d.numbers() {
			q := d.last
			if relflag {
				q.X += n
			} else {
				q.X = n
			}
			d.addcmd('L', q)
		}

	case 'V':
		// vertical lines
		for _, n := range d.numbers() {
			q := d.last
			if relflag {
				q.Y += n
			} else {
				q.Y = n
			}
			d.addcmd('L', q)
		}

	case 'C':
		// cubic Bézier
		startp := d.pos
		v := d.points()
		d.cubicBézier(startp, v, rel)

	case 'S':
		// smooth cubic Bézier
		startp := d.pos
		v := d.points()
		if len(v) < 2 {
			d.seterrf("Invalid smooth cubic Bézier at %d", startp)
			return
		}

		var p1, p2, p3 Point
		p1.X = 2*d.last.X - d.lastc.X
		p1.Y = 2*d.last.Y - d.lastc.Y
		p2 = rel(v[0])
		p3 = rel(v[1])

		d.addcmd('C', p1, p2, p3)
		d.cubicBézier(startp, v[2:], rel)

	case 'Q':
		// quadratic Bézier
		startp := d.pos
		v := d.points()
		d.quadraticBézier(startp, v, rel)

	case 'T':
		// smooth quadratic Bézier
		startp := d.pos
		v := d.points()
		if len(v) < 1 {
			d.seterrf("Invalid smooth cubic Bézier at %d", startp)
			return
		}

		var p1, p2 Point
		p1.X = 2*d.last.X - d.lastq.X
		p1.Y = 2*d.last.Y - d.lastq.Y
		p2 = rel(v[0])
		d.addcmd('Q', p1, p2)
		d.quadraticBézier(startp, v[1:], rel)

	case 'A':
		startp := d.pos
		v := d.numbers()
		if len(v) == 0 && len(v)%7 != 0 {
			d.seterrf("Invalid arc at %d", startp)
			return
		}

		for i := 0; i < len(v); i += 7 {
			w := v[i : i+7]
			r := Point{w[0], w[1]}
			angle := w[2]
			var largeArc, sweep bool
			if w[3] != 0 {
				largeArc = true
			}
			if w[4] != 0 {
				sweep = true
			}
			c := rel(Point{w[5], w[6]})
			pts := arcToBezier(d.last, c, r, angle, largeArc, sweep)
			d.addcmd('C', pts...)
			d.lastc = d.last
		}

	case 'Z':
		d.last = d.first
		d.lastc = d.last
		d.lastq = d.last
	}
}

func (d *pathdecoder) ellipticArc(rx, ry, angle float64, largeArc, sweep bool, p2 Point) {
	p1 := d.last
	if p1 == p2 {
		return
	}
}

func (d *pathdecoder) addcmd(cmd byte, v ...Point) {
	previ := len(d.cmd) - 1
	if cmd != 'M' && previ >= 0 && d.cmd[previ].Cmd == cmd {
		// append coords to last command
		prevc := &d.cmd[previ]
		prevc.Pt = append(prevc.Pt, v...)
	} else {
		// add new command
		d.cmd = append(d.cmd, PathCmd{
			Cmd: cmd,
			Pt:  v,
		})
	}

	d.last = v[len(v)-1]

	if cmd == 'M' {
		d.first = d.last
	}

	if cmd == 'C' {
		d.lastc = v[len(v)-2]
	} else {
		d.lastc = d.last
	}

	if cmd == 'Q' {
		d.lastq = v[len(v)-2]
	} else {
		d.lastq = d.last
	}
}

func (d *pathdecoder) line(v []Point, rel func(Point) Point) {
	for _, p := range v {
		d.addcmd('L', rel(p))
	}
}

func (d *pathdecoder) cubicBézier(startp int,
	v []Point, rel func(Point) Point) {

	if len(v)%3 != 0 {
		d.seterrf("Invalid number of cubic Bézier coords at %d", startp)
		return
	}

	for i := 0; i < len(v); i += 3 {
		s := v[i : i+3]
		p1 := rel(s[0])
		p2 := rel(s[1])
		p3 := rel(s[2])

		d.addcmd('C', p1, p2, p3)
	}
}

func (d *pathdecoder) quadraticBézier(startp int,
	v []Point, rel func(Point) Point) {

	if len(v)%2 != 0 {
		d.seterrf("Invalid number of quadratic Bézier coords at %d", startp)
		return
	}

	for i := 0; i < len(v); i += 2 {
		s := v[i : i+2]
		p1 := rel(s[0])
		p2 := rel(s[1])

		d.addcmd('Q', p1, p2)
	}
}

func (d *pathdecoder) skipspace() {
	for d.pos < len(d.data) && d.data[d.pos] == ' ' {
		d.pos++
	}
}

func (d *pathdecoder) isnum() bool {
	d.skipspace()
	if d.pos == len(d.data) {
		return false
	}

	return isnumbyte(d.data[d.pos])
}

func (d *pathdecoder) nextcmd() (cmd byte, relflag bool) {
	d.skipspace()

	if d.pos == len(d.data) {
		d.seterr(io.ErrUnexpectedEOF)
		return 0, false
	}

	c := d.data[d.pos]
	if strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", c) < 0 {
		d.seterr(fmt.Errorf("Unexpected command %q at position %d", c, d.pos))
		return 0, false
	}

	d.pos++

	relflag = c >= 'a'
	if relflag {
		c -= 'a' - 'A'
	}

	return c, relflag
}

func (d *pathdecoder) skipcomma() {
	d.skipspace()

	if d.pos == len(d.data) {
		return
	}

	if c := d.data[d.pos]; c == ',' {
		d.pos++
	}

	d.skipspace()
}

func (d *pathdecoder) numbers() []float64 {
	r := []float64{d.number()}
	for d.err == nil && d.isnum() {
		r = append(r, d.number())
	}
	return r
}

func (d *pathdecoder) number() float64 {
	d.skipspace()

	s := d.pos
	e := s

	for e < len(d.data) && isnumbyte(d.data[e]) {
		e++
	}

	if s == e {
		d.seterrf("Expected number at position %d", s)
		return 0
	}

	v, err := strconv.ParseFloat(d.data[s:e], 64)
	if err != nil {
		d.seterrf("Invalid number at position %d: %w", s, err)
		return 0
	}

	d.pos = e

	d.skipcomma()

	return v
}

func (d *pathdecoder) points() []Point {
	r := []Point{d.point()}
	for d.err == nil && d.isnum() {
		r = append(r, d.point())
	}
	return r
}

func (d *pathdecoder) point() Point {
	x := d.number()
	y := d.number()
	return Point{x, y}
}

func (d *pathdecoder) seterrf(format string, args ...interface{}) {
	d.seterr(fmt.Errorf(format, args...))
}

func (d *pathdecoder) seterr(err error) {
	if d.err == nil {
		d.err = err
	}
}

func isnumbyte(c byte) bool {
	return c == '.' || c == '-' || ('0' <= c && c <= '9') || c == 'e' || c == 'E'
}
