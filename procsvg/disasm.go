package main

import (
	"fmt"
	"image/color"
	"io"
)

func DumpPack(r io.Reader, w io.Writer) {
	var header [8]byte
	_, err := io.ReadFull(r, header[:])
	if err != nil {
		fmt.Fprintf(w, "# I/O ERROR %s", err)
		return
	}

	if string(header[:4]) != PackMagic {
		fmt.Fprintf(w, "# invalid header %02x\n", header[:4])
		return
	}

	nicons := int(byteOrder.Uint32(header[4:]))
	fmt.Fprintf(w, "# %d icons\n", nicons)

	var pal0 []color.NRGBA

	for {
		magic, data, err := readSection(r)
		if err == io.EOF {
			fmt.Fprintln(w, "# EOF")
			return
		}
		if err != nil {
			fmt.Fprintf(w, "# I/O ERROR %s", err)
			return
		}

		switch magic {

		case PaletteMagic:
			pal, idx, err := parsePalette(data)
			if err != nil {
				fmt.Fprintf(w, "# palette data ERROR %s", err)
				return
			}
			fmt.Fprintf(w, "PALETTE %d # %d entries\n", idx, len(pal))
			for i, c := range pal {
				fmt.Fprintf(w, "%02x %02x %02x %02x  RGBA %3d: %s\n",
					c.R, c.G, c.B, c.A, i, colorstr(c))
			}
			if idx == 0 {
				pal0 = pal
			}
			fmt.Fprintln(w)

		case IconMagic:
			pe, err := parseIcon(data)
			if err != nil {
				fmt.Fprintf(w, "# icon data ERROR %s", err)
				return
			}

			for _, m := range pe.Image {
				fmt.Fprintf(w, "ICON %q %d×%d\n", pe.Name, m.Width, m.Height)
				disasm(w, pal0, m.Data)
				fmt.Fprintln(w)
			}

		default:
			fmt.Fprintf(w, "# unrecognised section '%s'\n\n", magic)
		}
	}
}

func readSection(r io.Reader) (magic string, data []byte, err error) {
	var header [8]byte
	_, err = io.ReadFull(r, header[:])
	if err != nil {
		return "", nil, err
	}

	nbytes := int(byteOrder.Uint32(header[4:]))
	if nbytes > 1<<20 {
		return "", nil, fmt.Errorf("Section size too large")
	}

	data = make([]byte, int(nbytes))
	if _, err := io.ReadFull(r, data); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		return "", nil, err
	}

	return string(header[:4]), data, nil
}

func parsePalette(data []byte) (pal []color.NRGBA, idx int, err error) {
	n := len(data)
	if n < 2 || n != 2+4*int(data[1]) {
		return nil, 0, fmt.Errorf("invalid palette size %d", n)
	}

	idx = int(data[0])
	for i := 2; i < n; i += 4 {
		c := data[i:]
		pal = append(pal, color.NRGBA{c[0], c[1], c[2], c[3]})
	}
	return pal, idx, nil
}

var errInvalidIconHeader = fmt.Errorf("Invalid icon header")

func parseIcon(data []byte) (PackElem, error) {
	e := int(data[0]) + 1
	if e+1 > len(data) {
		return PackElem{}, errInvalidIconHeader
	}

	name := string(data[1:e])

	numimages := int(data[e])
	data = data[e+1:]

	const ihbytes = 8
	if len(data) < numimages*ihbytes {
		return PackElem{}, errInvalidIconHeader
	}

	type imageInfo struct {
		dx, dy   uint16
		bytesize int
	}

	var ii []imageInfo
	for i := 0; i < numimages; i++ {
		dx := byteOrder.Uint16(data[0:])
		dy := byteOrder.Uint16(data[2:])
		bs := int(byteOrder.Uint32(data[4:]))

		data = data[8:]

		ii = append(ii, imageInfo{dx, dy, bs})
	}

	pe := PackElem{Name: name}
	for _, inf := range ii {
		if len(data) < inf.bytesize {
			return PackElem{}, fmt.Errorf("Image data invalid")
		}

		pe.Image = append(pe.Image, &ProgImage{
			Width:  int(inf.dx),
			Height: int(inf.dy),
			Data:   data[:inf.bytesize],
		})

		data = data[inf.bytesize:]
	}

	if len(data) != 0 {
		return PackElem{}, fmt.Errorf("Garbage after image data")
	}

	return pe, nil
}

type ProgReader struct {
	out     io.Writer
	data    []byte
	pal     []color.NRGBA
	pos     int
	invalid bool
}

func (r *ProgReader) Byte() byte {
	if r.pos >= len(r.data) {
		return 0
	}

	c := r.data[r.pos]
	r.pos++
	return c
}

func (r *ProgReader) Point() {
	s := r.pos
	x, y := r.Coord(), r.Coord()
	e := r.pos
	dump := fmt.Sprintf("% 02x", r.data[s:e])
	fmt.Fprintf(r.out, "%-24s   %8.4f  %8.4f\n", dump, x, y)
}

func (r *ProgReader) Coord() float64 {
	c, n := CoordFromBytes(r.data[r.pos:])
	r.pos += n
	return c
}

func (pr *ProgReader) step() {
	if pr.pos == len(pr.data) {
		return
	}

	if !pr.invalid {
		pr.stepCmd()
		return
	}

	e := pr.pos + 8
	if e > len(pr.data) {
		e = len(pr.data)
	}

	dump := fmt.Sprintf("% 02x", pr.data[pr.pos:e])
	fmt.Fprintln(pr.out, dump)

	pr.pos = e
}

func colorstr(c color.NRGBA) string {
	if c.A != 255 {
		return fmt.Sprintf("rgba(%.4f,%.4f,%.4f,%.4f)",
			float64(c.R)/255,
			float64(c.G)/255,
			float64(c.B)/255,
			float64(c.A)/255)
	} else {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
}

func (pr *ProgReader) stepCmd() {
	s := pr.pos
	op := pr.Byte()
	var cmd string
	var ncoords int
	switch op & 0xf0 {

	case 0x00:
		switch op {

		case 0x00:
			cmd = "STOP"

		case 0x01:
			c := color.NRGBA{pr.Byte(), pr.Byte(), pr.Byte(), pr.Byte()}
			cmd = fmt.Sprintf("SOLIDFILL-rgba %s", colorstr(c))

		case 0x02:
			i := int(pr.Byte())
			var c color.NRGBA
			if i < len(pr.pal) {
				c = pr.pal[i]
			} else {
				fmt.Fprintln(pr.out, "# INVALID palette index")
			}
			cmd = fmt.Sprintf("SOLIDFILL-idx %d → %s", i, colorstr(c))
		}

	case 0x70:
		if op == 0x70 {
			ncoords = 1
			cmd = "M-begin"
		}
		if op == 0x71 {
			ncoords = 1
			cmd = "M-cont"
		}

	case 0x80, 0x90:
		ncoords = int(op-0x80) + 1
		cmd = fmt.Sprintf("L %d", ncoords)

	case 0xa0:
		nseg := int(op-0xa0) + 1
		ncoords = 3 * nseg
		cmd = fmt.Sprintf("C %d", nseg)

	case 0xb0:
		nseg := int(op-0xb0) + 1
		ncoords = 2 * nseg
		cmd = fmt.Sprintf("Q %d", nseg)
	}

	if cmd == "" {
		pr.invalid = true
		cmd = "INVALID"
	}

	e := pr.pos
	dump := fmt.Sprintf("% 02x", pr.data[s:e])
	fmt.Fprintf(pr.out, "%-24s  %s\n", dump, cmd)
	for i := 0; i < ncoords; i++ {
		pr.Point()
	}

	return
}

func disasm(w io.Writer, pal []color.NRGBA, data []byte) {
	r := ProgReader{
		out:  w,
		data: data,
		pal:  pal,
	}
	fmt.Fprintln(r.out, "# viewbox:")
	r.Point()
	r.Point()

	for r.pos < len(r.data) {
		r.step()
	}
}
