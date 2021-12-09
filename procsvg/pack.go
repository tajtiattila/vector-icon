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
	icons, err := findIcons(project)
	if err != nil {
		return err
	}

	colorStats := make(map[color.NRGBA]int)
	for _, icon := range icons {
		for _, fn := range icon.path {
			if err := CollectSvgColors(fn, colorStats); err != nil {
				return err
			}
		}
	}

	vpal := getpalv(project, colorStats)

	var pal0 []color.NRGBA
	if len(vpal) != 0 {
		pal0 = vpal[0]
	}

	var pev []PackElem
	for _, icon := range icons {
		pe := PackElem{Name: icon.name}
		for _, fn := range icon.path {
			if cli.verbose {
				fmt.Fprintf(os.Stderr, "Packing %s\n", fn)
			}
			x, err := ConvertSvg(fn, project.Epsilon, pal0)
			if err != nil {
				return fmt.Errorf("error converting %s: %w", fn, err)
			}
			pe.Image = append(pe.Image, x)
		}
		pev = append(pev, pe)
	}

	k := IconPack{
		palette: vpal,
	}
	for _, pe := range pev {
		k.Add(pe)
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

		// find non-palette colors
		palm := make(map[color.NRGBA]struct{})
		for _, c := range pal0 {
			palm[c] = struct{}{}
		}
		var vcol []color.NRGBA
		for c := range colorStats {
			if _, ok := palm[c]; !ok {
				vcol = append(vcol, c)
			}
		}
		sort.Slice(vcol, func(i, j int) bool {
			return lessColor(vcol[i], vcol[j])
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

type iconFile struct {
	name string
	path []string
}

func findIcons(project Project) ([]iconFile, error) {
	m := make(map[string][]string)
	for _, sub := range project.SizeDir {
		sdir := filepath.Join(project.IconDir, sub)
		dir := filepath.Join(project.IntermediateDir, sub)
		svgs, err := readdirnames(sdir, "*.svg")
		if err != nil {
			return nil, err
		}

		for _, fn := range svgs {
			name := strings.TrimSuffix(fn, ".svg")

			path := filepath.Join(dir, fn)
			m[name] = append(m[name], path)
		}
	}

	var v []iconFile
	for k, pv := range m {
		v = append(v, iconFile{
			name: k,
			path: pv,
		})
	}
	sort.Slice(v, func(i, j int) bool {
		return v[i].name < v[j].name
	})

	return v, nil
}

func getpalv(project Project, colorStats map[color.NRGBA]int) [][]color.NRGBA {

	if len(project.Palette) != 0 {
		var p0 []color.NRGBA
		for _, c := range project.Palette {
			p0 = append(p0, color.NRGBA(c))
		}
		return applyTransforms(project, p0)
	}

	if !project.AutoPalette {
		return nil
	}

	type colFreq struct {
		c color.NRGBA
		n int
	}
	var cf []colFreq
	for c, n := range colorStats {
		cf = append(cf, colFreq{c, n})
	}

	sort.Slice(cf, func(i, j int) bool {
		if d := cf[i].n - cf[j].n; d != 0 {
			return d < 0
		}
		return lessColor(cf[i].c, cf[j].c)
	})

	var p0 []color.NRGBA
	for _, c := range cf {
		p0 = append(p0, c.c)
	}
	return applyTransforms(project, p0)
}

func lessColor(ci, cj color.NRGBA) bool {
	if ci.R != cj.R {
		return ci.R < cj.R
	}
	if ci.G != cj.G {
		return ci.G < cj.G
	}
	return ci.B < cj.B
}

func applyTransforms(project Project, p0 []color.NRGBA) [][]color.NRGBA {
	pv := [][]color.NRGBA{p0}
	for _, tr := range project.ColorTransform {
		var px []color.NRGBA
		for _, c := range p0 {
			c1, ok := tr.Map[c]
			if !ok {
				if tr.InvertYPrime {
					y, cb, cr := color.RGBToYCbCr(c.R, c.G, c.B)
					y = 255 - y
					c1.R, c1.G, c1.B = color.YCbCrToRGB(y, cb, cr)
					c1.A = c.A
				} else {
					c1 = c
				}
			}
			px = append(px, c1)
		}
		pv = append(pv, px)
	}
	return pv
}

type IconPack struct {
	palette [][]color.NRGBA
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

	for i, p := range k.palette {
		if err = writePalette(w, i, p); err != nil {
			return w.n, err
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

func writePalette(w io.Writer, idx int, pal []color.NRGBA) error {
	fmt.Fprint(w, PaletteMagic)

	buf := new(bytes.Buffer)
	buf.WriteByte(byte(idx))
	buf.WriteByte(byte(len(pal)))
	for _, c := range pal {
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
