package main

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
)

type ProgImage struct {
	Width  int
	Height int
	Data   []byte // Icon variant image
}

type ProgMem struct {
	buf bytes.Buffer

	// coordinate precision
	Precision float64

	inPath bool
	xform  Matrix
}

func NewProgMem(prec float64) *ProgMem {
	return &ProgMem{
		Precision: prec,
		xform:     MatrixIdentity,
	}
}

func (m *ProgMem) Bytes() []byte {
	return m.buf.Bytes()
}

func (m *ProgMem) Stop() {
	m.Byte(0)
}

func (m *ProgMem) Byte(b byte) {
	m.buf.WriteByte(b)
}

func (m *ProgMem) Color(c color.Color) {
	x := color.NRGBAModel.Convert(c).(color.NRGBA)
	m.Byte(x.R)
	m.Byte(x.G)
	m.Byte(x.B)
	m.Byte(x.A)
}

func (m *ProgMem) ViewBox(l, t, r, b float64) {
	m.Coord(l)
	m.Coord(t)
	m.Coord(r)
	m.Coord(b)
}

func (m *ProgMem) BeginPath(xform Matrix) {
	m.inPath = false
	m.xform = xform
}

func (m *ProgMem) PathCmd(c PathCmd) error {
	switch c.Cmd {
	case 'M':
		if len(c.Pt) != 1 {
			return fmt.Errorf("Move op with %d points", len(c.Pt))
		}
		if !m.inPath {
			m.Byte(0x70)
			m.inPath = true
		} else {
			m.Byte(0x71)
		}
		m.Pts(c.Pt)
		return nil

	case 'L':
		if len(c.Pt) == 0 {
			return fmt.Errorf("Empty line op")
		}
		return m.addOp(c.Cmd, 0x80, 0x20, 1, c.Pt)

	case 'C':
		mul := 3
		if n := len(c.Pt); n == 0 || n%mul != 0 {
			return fmt.Errorf("Empty or invalid cubic Bézier op length %d", n)
		}
		return m.addOp(c.Cmd, 0xa0, 0x10, mul, c.Pt)

	case 'Q':
		mul := 2
		if n := len(c.Pt); n == 0 || n%mul != 0 {
			return fmt.Errorf("Empty or invalid quadratic Bézier op length %d", n)
		}
		return m.addOp(c.Cmd, 0xb0, 0x10, mul, c.Pt)
	}

	return fmt.Errorf("Unknown path cmd %q", c.Cmd)
}

func (m *ProgMem) addOp(cmd byte, baseop byte, maxrep, mod int, pts []Point) error {
	n := len(pts)
	if n == 0 {
		return fmt.Errorf("Empty %c op", cmd)
	}
	if n%mod != 0 {
		return fmt.Errorf("Invalid %c op, length %d must be mod %d", cmd, n, mod)
	}

	rng := mod * maxrep
	for len(pts) > rng {
		m.Byte(baseop + byte(maxrep-1))
		m.Pts(pts[:rng])
		pts = pts[rng:]
	}
	m.Byte(baseop + byte(len(pts)/mod-1))
	m.Pts(pts)
	return nil
}

func (m *ProgMem) Pts(v []Point) {
	for _, p := range v {
		q := m.xform.Transform(p)
		m.Coord(q.X)
		m.Coord(q.Y)
	}
}

func (m *ProgMem) Coord(v float64) {
	var buf [4]byte
	n := CoordBytes(buf[:], v, m.Precision)
	m.buf.Write(buf[:n])
}

// CoordBytes puts coord v with precision prec in p,
// and returns the number of bytes used.
// It returns 0 if p is too small.
func CoordBytes(p []byte, v, prec float64) int {
	if len(p) == 0 {
		return 0
	}

	if i := math.Round(v); math.Abs(v-i) <= prec && i >= -64 && i < 64 {
		p[0] = (byte(int(i)+64) << 1) | 0x01
		return 1
	}

	if len(p) < 2 {
		return 0
	}

	if n := rd64(v); math.Abs(v-n) <= prec && n >= -128 && n < 128 {
		x := (uint16(int(n*64)+(128*64)) << 2) | 0x02
		p[0] = byte(x)
		p[1] = byte(x >> 8)
		return 2
	}

	if len(p) < 4 {
		return 0
	}

	bits := math.Float32bits(float32(v))
	p[0] = byte(bits) & ^byte(0x03)
	p[1] = byte(bits >> 8)
	p[2] = byte(bits >> 16)
	p[3] = byte(bits >> 24)

	return 4
}

func rd64(v float64) float64 {
	return math.Round(v*64) / 64
}

// CoordBytes reads coord v from p,
// and returns the number of bytes consumed.
// It returns n == 0 if p doesn't contain a full number.
func CoordFromBytes(p []byte) (v float64, n int) {
	if len(p) == 0 {
		return 0, 0
	}

	if (p[0] & 0x01) != 0 {
		return float64(int(p[0]>>1) - 64), 1
	}

	if len(p) < 2 {
		return 0, 0
	}

	if (p[0] & 0x02) != 0 {
		u := uint16(p[0]) | (uint16(p[1]) << 8)
		x := int(u>>2) - (128 * 64)
		return float64(x) / 64, 2
	}

	if len(p) < 4 {
		return 0, 0
	}

	var bits uint32
	bits |= uint32(p[0]) << 0
	bits |= uint32(p[1]) << 8
	bits |= uint32(p[2]) << 16
	bits |= uint32(p[3]) << 24

	return float64(math.Float32frombits(bits)), 4
}
