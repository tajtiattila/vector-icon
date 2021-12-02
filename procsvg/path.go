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
	c, rel := d.nextcmd()
	switch c {

	case 'M':
		pts := d.points()
		p0 := pts[0]
		if rel {
			d.last.X += p0.X
			d.last.Y += p0.Y
		} else {
			d.last = p0
		}
		d.first = d.last
		d.addcmd('M', d.last)
		d.line(pts[1:], rel)

	case 'L':
		d.line(d.points(), rel)

	case 'H':
		// horizontal lines
		for _, n := range d.numbers() {
			if rel {
				d.last.X += n
			} else {
				d.last.X = n
			}
			d.addcmd('L', d.last)
		}

	case 'V':
		// vertical lines
		for _, n := range d.numbers() {
			if rel {
				d.last.Y += n
			} else {
				d.last.Y = n
			}
			d.addcmd('L', d.last)
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
		if rel {
			p2.X = d.last.X + v[0].X
			p2.Y = d.last.Y + v[0].Y
			p3.X = d.last.X + v[1].X
			p3.Y = d.last.Y + v[1].Y
		} else {
			p2, p3 = v[0], v[1]
		}

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
		if rel {
			p2.X = d.last.X + v[0].X
			p2.Y = d.last.Y + v[0].Y
		} else {
			p2 = v[0]
		}
		d.addcmd('Q', p1, p2)
		d.quadraticBézier(startp, v[1:], rel)

	case 'A':
		d.seterrf("Elliptic arc not implemented at %d", d.pos)

	case 'Z':
		if d.first != d.last {
			d.addcmd('L', d.first)
		}
	}
}

func (d *pathdecoder) addcmd(cmd byte, v ...Point) {
	lasti := len(d.cmd) - 1
	if cmd != 'M' && lasti >= 0 && d.cmd[lasti].Cmd == cmd {
		// append coords to last command
		lastc := &d.cmd[lasti]
		lastc.Pt = append(lastc.Pt, v...)
	} else {
		// add new command
		d.cmd = append(d.cmd, PathCmd{
			Cmd: cmd,
			Pt:  v,
		})
	}

	d.last = v[len(v)-1]

	if cmd == 'C' {
		d.lastc = v[1]
	} else {
		d.lastc = d.last
	}

	if cmd == 'Q' {
		d.lastq = v[0]
	} else {
		d.lastq = d.last
	}
}

func (d *pathdecoder) line(v []Point, rel bool) {
	for _, p := range v {
		var q Point
		if rel {
			d.last.X += p.X
			d.last.Y += p.Y
		} else {
			q = p
		}
		d.addcmd('L', q)
	}
}

func (d *pathdecoder) cubicBézier(startp int, v []Point, rel bool) {
	if len(v)%3 != 0 {
		d.seterrf("Invalid number of cubic Bézier coords at %d", startp)
		return
	}

	for i := 0; i < len(v); i += 3 {
		s := v[i : i+3]
		var p1, p2, p3 Point
		if rel {
			p1.X = d.last.X + s[0].X
			p1.Y = d.last.Y + s[0].Y
			p2.X = d.last.X + s[1].X
			p2.Y = d.last.Y + s[1].Y
			p3.X = d.last.X + s[2].X
			p3.Y = d.last.Y + s[2].Y
		} else {
			p1, p2, p3 = s[0], s[1], s[2]
		}

		d.addcmd('C', p1, p2, p3)
	}
}

func (d *pathdecoder) quadraticBézier(startp int, v []Point, rel bool) {
	if len(v)%2 != 0 {
		d.seterrf("Invalid number of quadratic Bézier coords at %d", startp)
		return
	}

	for i := 0; i < len(v); i += 2 {
		s := v[i : i+2]
		var p1, p2 Point
		if rel {
			p1.X = d.last.X + s[0].X
			p1.Y = d.last.Y + s[0].Y
			p2.X = d.last.X + s[1].X
			p2.Y = d.last.Y + s[1].Y
		} else {
			p1, p2 = s[0], s[1]
		}

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

func (d *pathdecoder) nextcmd() (cmd byte, relative bool) {
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

	relative = c >= 'a'
	if relative {
		c -= 'a'
	}

	return c, relative
}

func (d *pathdecoder) comma() {
	d.skipspace()

	if d.pos == len(d.data) {
		d.seterr(io.ErrUnexpectedEOF)
		return
	}

	if c := d.data[d.pos]; c != ',' {
		d.seterr(fmt.Errorf("Expected comma ',' at position %d, got %q", d.pos, c))
		return
	}

	d.pos++
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
	d.comma()
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
	return c == '.' || c == '-' || ('0' <= c && c <= '9')
}
