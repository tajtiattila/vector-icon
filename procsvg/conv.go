package main

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	"os"
	"strconv"
	"strings"
)

type svgOpts struct {
	eps float64 // conversion precision

	palette     []color.NRGBA
	colorMagnet float64

	colorCount map[color.NRGBA]int
}

func ProcSvg(fn string, opts svgOpts) (*ProgImage, error) {
	svg, err := fileXmlTree(fn)
	if err != nil {
		return nil, err
	}

	cm := make(map[color.NRGBA]int)
	for i, c := range opts.palette {
		cm[c] = i
	}

	g := svgprog{
		fn:         fn,
		palette:    opts.palette,
		cmsquare:   opts.colorMagnet * opts.colorMagnet,
		colormap:   cm,
		colorCount: opts.colorCount,
	}
	g.mem.Precision = opts.eps

	err = g.tree(svg)

	return g.finish(), err
}

func fileXmlTree(fn string) (Node, error) {
	f, err := os.Open(fn)
	if err != nil {
		return Node{}, err
	}
	defer f.Close()

	node, err := xmlTree(f)
	if err != nil {
		return Node{}, err
	}

	return node, nil
}

type Node struct {
	Name xml.Name
	Attr []xml.Attr
	Node []Node // child nodes
}

func xmlTree(r io.Reader) (Node, error) {
	d := xml.NewDecoder(r)

	var root []Node
	var stack []*Node
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Node{}, err
		}

		switch x := tok.(type) {
		case xml.StartElement:
			var node Node
			node.Name = x.Name
			node.Attr = x.Attr
			var p *Node
			if n := len(stack); n == 0 {
				root = append(root, node)
				p = &root[len(root)-1]
			} else {
				parent := stack[n-1]
				parent.Node = append(parent.Node, node)
				p = &parent.Node[len(parent.Node)-1]
			}
			stack = append(stack, p)

		case xml.EndElement:
			n := len(stack)
			stack = stack[:n-1]
		}
	}

	switch len(root) {
	case 0:
		return Node{}, fmt.Errorf("Empty doc")
	case 1:
		return root[0], nil
	default:
		return Node{}, fmt.Errorf("Multiple root nodes")
	}
}

type svgprog struct {
	fn       string
	colormap map[color.NRGBA]int

	palette  []color.NRGBA
	cmsquare float64

	colorCount map[color.NRGBA]int

	im  *ProgImage
	mem ProgMem

	xform []Matrix

	solidFillColor color.NRGBA

	// non-palette colors
	colors []color.NRGBA
}

func (g *svgprog) finish() *ProgImage {
	g.mem.Stop()
	g.im.Data = g.mem.Bytes()
	return g.im
}

func (g *svgprog) tree(n Node) error {
	// Handle inherited attributes.
	if is_hidden(n) {
		return nil
	}
	if hasattr(n, "clip-path") {
		fmt.Fprintf(os.Stderr, "clip-path found in %s\n", g.fn)
	}
	if hasattr(n, "transform") {
		mat, err := SvgTransformMatrix(findattr(n, "transform"))
		if err != nil {
			return fmt.Errorf("transform: %w", err)
		}
		g.pushTransform(g.transform().Mul(mat))
		defer g.popTransform()
	}
	if fill, ok := get_svg_solid_fill(n); ok {
		oldFill := g.solidFillColor
		g.solidFillColor = fill
		defer func() {
			g.solidFillColor = oldFill
		}()
	}

	// Handle nodes of interest.
	var err error
	switch n.Name.Local {
	case "svg":
		err = g.svg(n)

	case "path":
		err = g.path(n)
	}
	if err != nil {
		return err
	}

	// Handle node children.
	for _, child := range n.Node {
		if err := g.tree(child); err != nil {
			return err
		}
	}

	return nil
}

func (g *svgprog) svg(n Node) error {
	g.im = new(ProgImage)

	var err error

	ws := findattr(n, "width")
	g.im.Width, err = strconv.Atoi(strings.TrimSuffix(ws, "px"))
	if err != nil {
		return fmt.Errorf("error parsing width %q", ws)
	}

	hs := findattr(n, "height")
	g.im.Height, err = strconv.Atoi(strings.TrimSuffix(hs, "px"))
	if err != nil {
		return fmt.Errorf("error parsing width %q", hs)
	}

	vb := findattr(n, "viewBox")
	var minx, miny, width, height float64
	_, err = fmt.Sscan(vb, &minx, &miny, &width, &height)
	if err != nil {
		return fmt.Errorf("error parsing viewBox %q", vb)
	}

	g.mem.ViewBox(minx, miny, minx+width, miny+height)
	return nil
}

func (g *svgprog) path(n Node) error {
	c := g.solidFillColor
	if c.A == 0 {
		if cli.verbose {
			fmt.Println("skipping invisible path")
		}
		return nil
	}

	cmds, err := PathDCmds(findattr(n, "d"))
	if err != nil {
		return err
	}

	if len(cmds) == 0 {
		return nil
	}

	if len(cmds) == 1 && cmds[0].Cmd == 'M' {
		return nil
	}

	g.handle_fill()

	g.mem.BeginPath(g.transform())
	for _, c := range cmds {
		if err := g.mem.PathCmd(c); err != nil {
			return err
		}
	}

	return nil
}

func (g *svgprog) transform() Matrix {
	n := len(g.xform)
	if n == 0 {
		return MatrixIdentity
	}
	return g.xform[n-1]
}

func (g *svgprog) pushTransform(m Matrix) {
	g.xform = append(g.xform, m)
}

func (g *svgprog) popTransform() {
	n := len(g.xform)
	g.xform = g.xform[:n-1]
}

func (g *svgprog) handle_fill() {
	c := g.solidFillColor

	if g.cmsquare > 0 {
		for _, x := range g.palette {
			dr := int(c.R) - int(x.R)
			dg := int(c.G) - int(x.G)
			db := int(c.B) - int(x.B)
			dsquare := float64(dr*dr + dg*dg + db*db)
			if dsquare < g.cmsquare {
				c = x
				break
			}
		}
	}

	if g.colorCount != nil {
		if cli.showColor {
			fmt.Printf("  #%02x%02x%02x\n", c.R, c.G, c.B)
		}
		g.colorCount[c]++
	}

	if i, ok := g.colormap[c]; ok {
		g.mem.Byte(0x02)
		g.mem.Byte(byte(i))
	} else {
		g.colors = append(g.colors, c)

		g.mem.Byte(0x01)
		g.mem.Color(c)
	}
}

func hasattr(n Node, name string) bool {
	for _, a := range n.Attr {
		if a.Name.Local == name {
			return true
		}
	}
	return false
}

func findattr(n Node, name string) string {
	var r string
	found := false
	for _, a := range n.Attr {
		if a.Name.Local == name {
			if found {
				panic(fmt.Errorf("duplicate attr %s", name))
			}
			r = a.Value
			found = true
		}
	}
	return r
}

func get_presentation_attr(n Node, attr string) string {
	style := cssdecode(findattr(n, "style"))
	if v, ok := style[attr]; ok {
		return v
	}

	return findattr(n, attr)
}

func get_svg_solid_fill(n Node) (color.NRGBA, bool) {
	a := get_presentation_attr(n, "fill")
	if a == "none" {
		return color.NRGBA{0, 0, 0, 0}, true
	}

	return colorfromhex(a)
}

func is_hidden(n Node) bool {
	a := get_presentation_attr(n, "display")
	return a == "none"
}

func colorfromhex(s string) (color.NRGBA, bool) {
	if s == "" || s[0] != '#' {
		return color.NRGBA{}, false
	}

	v, err := strconv.ParseUint(s[1:], 16, 64)
	if err != nil {
		return color.NRGBA{}, false
	}

	switch len(s) - 1 {
	case 3:
		r := (byte(v>>8) & 0xf) * 0x11
		g := (byte(v>>4) & 0xf) * 0x11
		b := (byte(v) & 0xf) * 0x11
		return color.NRGBA{r, g, b, 0xff}, true

	case 6:
		r := byte(v >> 16)
		g := byte(v >> 8)
		b := byte(v)
		return color.NRGBA{r, g, b, 0xff}, true
	}

	return color.NRGBA{}, false
}

func cssdecode(css string) map[string]string {
	if css == "" {
		return nil
	}

	v := strings.Split(css, ";")
	m := make(map[string]string)
	for _, e := range v {
		colon := strings.Index(e, ":")
		if colon > 0 {
			m[strings.TrimSpace(e[:colon])] = strings.TrimSpace(e[colon+1:])
		}
	}
	return m
}
