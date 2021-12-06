package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var byteOrder = binary.LittleEndian

func pack_project(project Project) error {
	m := make(map[string][]*ProgImage)
	mcol := make(map[color.NRGBA]struct{})
	for _, sub := range project.SizeDirs {
		sdir := filepath.Join(project.IconDir, sub)
		dir := filepath.Join(project.IntermediateDir, sub)
		svgs, err := readdirnames(sdir, "*.svg")
		if err != nil {
			return err
		}

		for _, fn := range svgs {
			name := strings.TrimSuffix(fn, ".svg")
			path := filepath.Join(dir, fn)
			if cli.verbose {
				fmt.Fprintf(os.Stderr, "Packing %s\n", path)
			}
			x, colors, err := ConvertSvg(path, project)
			if err != nil {
				return fmt.Errorf("error converting %s: %w", path, err)
			}
			m[name] = append(m[name], x)

			for _, c := range colors {
				mcol[c] = struct{}{}
			}
		}
	}

	var pe []PackElem
	for n, im := range m {
		pe = append(pe, PackElem{Name: n, Image: im})
	}

	sort.Slice(pe, func(i, j int) bool {
		return pe[i].Name < pe[j].Name
	})

	k := IconPack{
		palette: project.Palette,
	}
	for _, e := range pe {
		k.Add(e)
	}

	f, err := os.Create(project.Target)
	if err != nil {
		return err
	}
	defer f.Close()

	var w io.Writer
	if cli.disasm {
		fa, err := os.Create(project.Target + ".disasm")
		if err != nil {
			return err
		}
		defer fa.Close()

		var vcol []color.NRGBA
		for c := range mcol {
			vcol = append(vcol, c)
		}
		sort.Slice(vcol, func(i, j int) bool {
			ci := vcol[i]
			cj := vcol[j]
			if ci.R != cj.R {
				return ci.R < cj.R
			}
			if ci.G != cj.G {
				return ci.G < cj.G
			}
			return ci.B < cj.B
		})

		if len(vcol) != 0 {
			fmt.Fprintln(fa, "# non-palette image colors:")
			for _, c := range vcol {
				fmt.Fprintf(fa, "# %02x%02x%02x\n", c.R, c.G, c.B)
			}
			fmt.Fprintln(fa)
		}

		pr, pw := io.Pipe()
		go func() {
			DumpPack(pr, fa)
		}()
		defer pw.Close()

		w = io.MultiWriter(f, pw)
	} else {
		w = f
	}

	k.WriteTo(w)

	return nil
}

type IconPack struct {
	palette []ColorRow
	elem    []PackElem
}

type PackElem struct {
	Name  string
	Image []*ProgImage
}

func (e *PackElem) writeTo(w io.Writer) error {
	d := e.dataBytes()

	writeUint32(w, uint32(len(d)))

	_, err := w.Write(d)
	return err
}

func (e *PackElem) dataBytes() []byte {

	n := e.Name
	if len(n) > 255 {
		n = n[:255]
	}

	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(n)))
	buf.WriteString(n)
	buf.WriteByte(byte(len(e.Image)))

	const ihbytes = 8
	var ih [ihbytes]byte // image header
	for _, im := range e.Image {
		byteOrder.PutUint16(ih[0:], uint16(im.Width))
		byteOrder.PutUint16(ih[2:], uint16(im.Height))
		byteOrder.PutUint32(ih[4:], uint32(len(im.Data)))
		buf.Write(ih[:])
	}

	for _, im := range e.Image {
		buf.Write(im.Data)
	}

	return buf.Bytes()
}

func (k *IconPack) Add(pe PackElem) {
	// sort icons by size
	size := func(i int) int {
		return pe.Image[i].Width * pe.Image[i].Height
	}
	sort.Slice(pe.Image, func(i, j int) bool {
		return size(i) > size(j)
	})

	k.elem = append(k.elem, pe)
}

const PackMagic = "icpk"
const PaletteMagic = "PALT"
const IconMagic = "ICON"

func (k *IconPack) WriteTo(w0 io.Writer) (n int64, err error) {
	w := &countWriter{w: w0}

	fmt.Fprint(w, PackMagic)
	if _, err := writeUint32(w, uint32(len(k.elem))); err != nil {
		return w.n, err
	}

	if len(k.palette) != 0 {
		npals := 1
		for _, cr := range k.palette {
			if n := len(cr); n > npals {
				npals = n
			}
		}

		for i := 0; i < npals; i++ {
			if err = k.writePalette(w, i); err != nil {
				return w.n, err
			}
		}
	}

	for _, e := range k.elem {
		fmt.Fprint(w, IconMagic)
		err := e.writeTo(w)
		if err != nil {
			return w.n, err
		}
	}

	return w.n, nil
}

func (k *IconPack) writePalette(w io.Writer, i int) error {
	fmt.Fprint(w, PaletteMagic)

	buf := new(bytes.Buffer)
	buf.WriteByte(byte(i))
	buf.WriteByte(byte(len(k.palette)))
	for _, cr := range k.palette {
		var c color.NRGBA
		if i < len(cr) {
			c = cr[i]
		}
		buf.WriteByte(c.R)
		buf.WriteByte(c.G)
		buf.WriteByte(c.B)
		buf.WriteByte(c.A)
	}

	if _, err := writeUint32(w, uint32(len(buf.Bytes()))); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

func writeUint32(w io.Writer, v uint32) (n int, err error) {
	var buf [4]byte
	byteOrder.PutUint32(buf[:], v)
	return w.Write(buf[:])
}

type countWriter struct {
	w io.Writer
	n int64
}

func (cw *countWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}
