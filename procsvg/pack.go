package main

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type IconPack struct {
	elem []PackElem
}

type PackElem struct {
	Name  string
	Image []*ProgImage
}

func (e *PackElem) writeTo(w io.Writer) error {
	d := e.dataBytes()

	var buf [32]byte
	n := binary.PutUVarint(buf[:], uint64(len(v)))

	_, err := w.Write(buf[:n])
	if err != nil {
		return err
	}

	_, err = w.Write(d)
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

func (k *IconPack) AddSvg(basename, dirs ...string) error {
	var im []*ProgImage
	for _, d := range dirs {
		x, err := ConvertSvg(filepath.Join(d, basename))
		if err != nil {
			return err
		}

		im = append(im, x)
	}

	k.elem = append(k.elem, PackElem{
		Name:  strings.TrimSuffix(basename, ".svg"),
		Image: im,
	})

	return nil
}

func (k *IconPack) WriteTo(w0 io.Writer) (n int64, err error) {
	w := &countWriter{w: w0}

	fmt.Fprint(w, "icpk")
	writeUVarint(uint64(len(k.elem)))

	for _, e := range k.elem {
		err := e.writeTo(w)
		if err != nil {
			return w.n, err
		}
	}

	return w.n, nil
}

func writeUVarint(w io.Writer, v uint64) (n int, err error) {
	var buf [32]byte
	n := encoding.PutUVarint(buf[:], v)
	return w.Write(buf[:n])
}

type countWriter struct {
	w io.Writer
	n int64
}

func (cw *countWriter) Write(p []byte) (n int, err error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
}
