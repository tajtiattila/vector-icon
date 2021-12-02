package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func pack_project(project Project) error {
	m := make(map[string][]*ProgImage)
	for _, sub := range project.SizeDirs {
		sdir := filepath.Join(project.IconDir, sub)
		dir := filepath.Join(project.IntermediateDir, sub)
		svgs, err := readdirnames(sdir, "*.svg")
		if err != nil {
			return err
		}

		for _, fn := range svgs {
			name := strings.TrimSuffix(fn, "*svg")
			path := filepath.Join(dir, fn)
			if cli.verbose {
				fmt.Fprintf(os.Stderr, "Packing %s\n", path)
			}
			x, err := ConvertSvg(path)
			if err != nil {
				return fmt.Errorf("error converting %s: %w", path, err)
			}
			m[name] = append(m[name], x)
		}
	}

	var pe []PackElem
	for n, im := range m {
		pe = append(pe, PackElem{Name: n, Image: im})
	}

	sort.Slice(pe, func(i, j int) bool {
		return pe[i].Name < pe[j].Name
	})

	k := IconPack{}
	for _, e := range pe {
		k.Add(e)
	}

	f, err := os.Create(project.Target)
	if err != nil {
		return err
	}
	defer f.Close()

	k.WriteTo(f)

	return nil
}

type IconPack struct {
	elem []PackElem
}

type PackElem struct {
	Name  string
	Image []*ProgImage
}

func (e *PackElem) writeTo(w io.Writer) error {
	d := e.dataBytes()

	writeUvarint(w, uint64(len(d)))

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

	enc := binary.LittleEndian

	const ihbytes = 8
	var ih [ihbytes]byte // image header
	ofs := len(e.Image) * ihbytes
	for _, im := range e.Image {
		enc.PutUint16(ih[0:], uint16(im.Width))
		enc.PutUint16(ih[2:], uint16(im.Height))
		enc.PutUint32(ih[4:], uint32(ofs))
		buf.Write(ih[:])
		ofs += len(im.Data)
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

func (k *IconPack) AddSvg(basename string, dirs ...string) error {
	pe := PackElem{
		Name: strings.TrimSuffix(basename, ".svg"),
	}

	for _, d := range dirs {
		x, err := ConvertSvg(filepath.Join(d, basename))
		if err != nil {
			return err
		}

		pe.Image = append(pe.Image, x)
	}

	k.Add(pe)
	return nil
}

func (k *IconPack) WriteTo(w0 io.Writer) (n int64, err error) {
	w := &countWriter{w: w0}

	fmt.Fprint(w, "icpk")
	writeUvarint(w, uint64(len(k.elem)))

	for _, e := range k.elem {
		err := e.writeTo(w)
		if err != nil {
			return w.n, err
		}
	}

	return w.n, nil
}

func writeUvarint(w io.Writer, v uint64) (n int, err error) {
	var buf [32]byte
	n = binary.PutUvarint(buf[:], v)
	return w.Write(buf[:n])
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
