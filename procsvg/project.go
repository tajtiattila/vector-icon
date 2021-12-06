package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"strings"
)

type Project struct {
	// source svg icon dir relative to project file
	IconDir string `json:"icondir"`

	// size subdirs for icons with multiple sizes or levels of detail
	SizeDirs []string `json:"sizedirs"`

	// intermediate dir relative to project file
	IntermediateDir string `json:"intermediatedir"`

	// conversion precision
	Epsilon float64 `json:"epsilon"`

	// Palette defines optional palettes.
	//
	// The first color in each row encodes SVG colors,
	// and is written as the icon pack palette index = 0.
	// Additional colors are written into additional palettes,
	// and may be used to encode styled/themed colors.
	Palette []ColorRow

	// target relative to project file
	Target string `json:"target"`
}

var DefaultProject = Project{
	IconDir:         "icons",
	IntermediateDir: "intermediate",
	SizeDirs:        []string{"."},
	Epsilon:         1e-4,
	Target:          "icons.iconpk",
}

func (p Project) ColorMap() (map[color.NRGBA]int, error) {
	if len(p.Palette) > 127 {
		return nil, fmt.Errorf("extended palette not implemented (max colors == 127)")
	}

	var nlen int
	m := make(map[color.NRGBA]int)
	for i, cr := range p.Palette {
		c := cr[0]
		m[c] = i
		if i == 0 {
			nlen = len(cr)
		} else {
			if len(cr) != nlen {
				return nil, fmt.Errorf("palette row lengths differ at %d", i)
			}
		}
	}

	return m, nil
}

type ColorRow []color.NRGBA

func (cr *ColorRow) UnmarshalJSON(p []byte) error {
	var s string
	if err := json.Unmarshal(p, &s); err != nil {
		return err
	}

	var v []color.NRGBA
	for _, elem := range strings.Split(s, " ") {
		if elem == "" {
			continue
		}
		c, ok := colorfromhex(elem)
		if !ok {
			return fmt.Errorf("Invalid color %s in palette", elem)
		}
		v = append(v, c)
	}

	if len(v) == 0 {
		return fmt.Errorf("Empty palette entry")
	}

	*cr = v
	return nil
}
