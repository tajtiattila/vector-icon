package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Project represents a procsvg project.
type Project struct {
	// source svg icon dir relative to project file
	IconDir string

	// size subdirs for icons with multiple sizes or levels of detail
	SizeDir []string

	// IntermediateDir relative to project file for preprocessing
	// SVG files using inkcape.
	IntermediateDir string

	// NameFormat is an optional fmt.Printf format to generate
	// process icon names and ID strings from the file name.
	NameFormat string

	// Epsilon is the icon conversion precision.
	Epsilon float64

	// Palette defines a default palette.
	// When colors are specified, they appear
	// at the beginning of the icon pack palette.
	Palette []ProjectColor

	// AutoPalette generates the palette automatically or
	// extends the default palette with colors used by the icons.
	AutoPalette bool

	// ColorTransform defines color transformations.
	// Each transformation yields a new palette.
	ColorTransform []ColorTransform

	// target relative to project file
	Target string

	// source file to generate
	GenerateSource []GenSrc
}

// DefaultProject contains default project attributes.
var DefaultProject = Project{
	IconDir:         "icons",
	IntermediateDir: "intermediate",
	SizeDir:         []string{"."},
	Epsilon:         1e-4,
	Target:          "icons.iconpk",
}

// GenSrc holds source generation details.
type GenSrc struct {
	Path     string
	Template string

	// IDPrefix is an optional prefix for generated IDs.
	IDPrefix string

	// BaseIndex is the index assigned to the first icon.
	BaseIndex int
}

// TemplateData is the template data used by GenSrc.Template.
type TemplateData struct {
	BaseIndex int // GenSrc.BaseIndex

	Icons []TemplateIcon // icons from the generated icon pack
}

type TemplateIcon struct {
	ID     string // identifier generated from the name
	Name   string // icon name
	Quoted string // quoted name
	Index  int    // icon index
}

// Padding returns a string of spaces needed pads ID to n runes.
func (i TemplateIcon) Padding(n int) string {
	nrunes := 0
	for range i.ID {
		nrunes++
	}
	npad := n - nrunes
	if npad > 0 {
		return strings.Repeat(" ", npad)
	}
	return ""
}

type ProjectColor color.NRGBA

func (c *ProjectColor) UnmarshalText(p []byte) error {
	s := string(p)

	if x, ok := colorfromhex(s); ok {
		*c = ProjectColor(x)
		return nil
	}

	return fmt.Errorf("Invalid color %s", s)
}

type ColorTransform struct {
	// Map defines mapped color pairs (old, new)
	Map ColorMap

	// InvertYPrime specifies that colors not present in the color map
	// shoudl have their Y' value inverted in the Y'CbCr color space.
	InvertYPrime bool
}

type ColorMap map[color.NRGBA]color.NRGBA

func (cm *ColorMap) UnmarshalText(p []byte) error {
	v := strings.FieldsFunc(string(p), func(r rune) bool {
		return !hexColorRune(r)
	})
	if len(v)%2 != 0 {
		return fmt.Errorf("invalid number of colormap colors: %d", len(v))
	}

	m := make(ColorMap)
	for i := 0; i < len(v); i += 2 {
		c0, ok := colorfromhex(v[i])
		if !ok {
			return fmt.Errorf("invalid color %s", v[i])
		}
		c1, ok := colorfromhex(v[i+1])
		if !ok {
			return fmt.Errorf("invalid color %s", v[i+1])
		}
		m[c0] = c1
	}
	*cm = m
	return nil
}

func hexColorRune(r rune) bool {
	return r == '#' ||
		('0' <= r && r <= '9') ||
		('a' <= r && r <= 'f') ||
		('A' <= r && r <= 'F')
}

func LoadProject(fn string) (Project, error) {
	p := DefaultProject

	f, err := os.Open(fn)
	if err != nil {
		return p, err
	}
	defer f.Close()

	_, err = toml.NewDecoder(f).Decode(&p)
	return p, err
}
