package main

import (
	"encoding/binary"
	"fmt"
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

	for i := 0; i < nicons; i++ {
		pe, err := readPackElem(r)
		if err != nil {
			fmt.Fprintf(w, "# I/O ERROR %s", err)
			return
		}

		for _, m := range pe.Image {
			fmt.Fprintf(w, "ICON %q %d×%d\n", pe.Name, m.Width, m.Height)
			disasm(w, m.Data)
		}
	}
}

func readPackElem(r io.Reader) (PackElem, error) {
	var fs uint32
	if err := binary.Read(r, byteOrder, &fs); err != nil {
		return PackElem{}, err
	}

	if fs > 1<<20 {
		return PackElem{}, fmt.Errorf("Invalid icon file size")
	}

	data := make([]byte, int(fs))
	if _, err := io.ReadFull(r, data); err != nil {
		return PackElem{}, err
	}

	invh := fmt.Errorf("Invalid icon header")
	if len(data) == 0 {
		return PackElem{}, invh
	}

	e := int(data[0]) + 1
	if e+1 > len(data) {
		return PackElem{}, invh
	}

	name := string(data[1:e])

	numimages := int(data[e])
	data = data[e+1:]

	const ihbytes = 8
	if len(data) < numimages*ihbytes {
		return PackElem{}, invh
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

func (pr *ProgReader) stepCmd() {
	s := pr.pos
	op := pr.Byte()
	var cmd string
	var ncoords int
	switch op & 0xf0 {

	case 0x00:
		if op == 0x00 {
			cmd = "STOP"
		} else if op == 0x01 {
			r, b, g, a := pr.Byte(), pr.Byte(), pr.Byte(), pr.Byte()
			var color string
			if a != 255 {
				color = fmt.Sprintf("rgba(%.4f, %.4f, %.4f, %.4f)",
					float64(r)/255,
					float64(g)/255,
					float64(b)/255,
					float64(a)/255)
			} else {
				color = fmt.Sprintf("#%02x%02x%02x", r, g, b)
			}
			cmd = fmt.Sprintf("SOLIDFILL %s", color)
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

func disasm(w io.Writer, data []byte) {
	r := ProgReader{
		out:  w,
		data: data,
	}
	fmt.Fprintln(r.out, "# viewbox:")
	r.Point()
	r.Point()

	for r.pos < len(r.data) {
		r.step()
	}
}
