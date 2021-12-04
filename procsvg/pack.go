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

var byteOrder = binary.LittleEndian

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
			name := strings.TrimSuffix(fn, ".svg")
			path := filepath.Join(dir, fn)
			if cli.verbose {
				fmt.Fprintf(os.Stderr, "Packing %s\n", path)
			}
			x, err := ConvertSvg(path, project.Epsilon)
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

func (k *IconPack) WriteTo(w0 io.Writer) (n int64, err error) {
	w := &countWriter{w: w0}

	fmt.Fprint(w, "icpk")
	writeUint32(w, uint32(len(k.elem)))

	for _, e := range k.elem {
		err := e.writeTo(w)
		if err != nil {
			return w.n, err
		}
	}

	return w.n, nil
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
