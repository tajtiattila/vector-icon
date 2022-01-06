package main

import (
	"fmt"
	"math"
	"strconv"
)

// Matrix represents a transformation matrix.
//
// Its six values [0..5] are form the matrix below:
//   [0] [2] [4]   m₁₁ m₁₂ m₁₃
//   [1] [3] [5]   m₂₁ m₂₂ m₂₃
//    0   0   1    m₃₁ m₃₂ m₃₃
type Matrix [6]float64

var MatrixIdentity = Matrix{1, 0, 0, 1, 0, 0}

func (m Matrix) Transform(p Point) Point {
	return Point{
		X: p.X*m[0] + p.Y*m[2] + m[4],
		Y: p.X*m[1] + p.Y*m[3] + m[5],
	}
}

// c₁₁ = a₁₁b₁₁ + a₁₂b₂₁ + a₁₃b₃₁
// c₂₁ = a₂₁b₁₁ + a₂₂b₂₁ + a₂₃b₃₁
// c₃₁ = a₃₁b₁₁ + a₃₂b₂₁ + a₃₃b₃₁ == 0
// c₁₂ = a₁₁b₁₂ + a₁₂b₂₂ + a₁₃b₃₂
// c₂₂ = a₂₁b₁₂ + a₂₂b₂₂ + a₂₃b₃₂
// c₃₂ = a₃₁b₁₂ + a₃₂b₂₂ + a₃₃b₃₂ == 0
// c₁₃ = a₁₁b₁₃ + a₁₂b₂₃ + a₁₃b₃₃
// c₂₃ = a₂₁b₁₃ + a₂₂b₂₃ + a₂₃b₃₃
// c₃₃ = a₃₁b₁₃ + a₃₂b₂₃ + a₃₃b₃₃ == 1

func (a Matrix) Mul(b Matrix) Matrix {
	// c₁₁ = a₁₁b₁₁ + a₁₂b₂₁
	// c₂₁ = a₂₁b₁₁ + a₂₂b₂₁
	// c₁₂ = a₁₁b₁₂ + a₁₂b₂₂
	// c₂₂ = a₂₁b₁₂ + a₂₂b₂₂
	// c₁₃ = a₁₁b₁₃ + a₁₂b₂₃ + a₁₃
	// c₂₃ = a₂₁b₁₃ + a₂₂b₂₃ + a₂₃

	return Matrix{
		a[0]*b[0] + a[2]*b[1],        // c₁₁
		a[1]*b[0] + a[3]*b[1],        // c₂₁
		a[0]*b[2] + a[2]*b[3],        // c₁₂
		a[1]*b[2] + a[3]*b[3],        // c₂₂
		a[0]*b[4] + a[2]*b[5] + a[4], // c₁₃
		a[1]*b[4] + a[3]*b[5] + a[5], // c₁₃
	}
}

func (m Matrix) Translate(cx, cy float64) Matrix {
	// b:
	//    1   0  cx    m₁₁ m₁₂ m₁₃
	//    0   1  cy    m₂₁ m₂₂ m₂₃
	//    0   0   1    m₃₁ m₃₂ m₃₃

	// c₁₁ = a₁₁
	// c₂₁ = a₂₁
	// c₁₂ = a₁₂
	// c₂₂ = a₂₂
	// c₁₃ = a₁₁cx + a₁₂cy + a₁₃
	// c₂₃ = a₂₁cx + a₂₂cy + a₂₃
	return Matrix{
		m[0],
		m[1],
		m[2],
		m[3],
		m[0]*cx + m[2]*cy + m[4],
		m[1]*cx + m[3]*cy + m[5],
	}
}

func (m Matrix) Rotate(θ float64) Matrix {
	// b:
	//  cosθ -sinθ  0    m₁₁ m₁₂ m₁₃
	//  sinٰθ  cosθ  0    m₂₁ m₂₂ m₂₃
	//    0     0   1    m₃₁ m₃₂ m₃₃

	sinθ, cosθ := math.Sincos(θ * math.Pi / 180)

	// c₁₁ = a₁₁cosθ + a₁₂sinθ
	// c₂₁ = a₂₁cosθ + a₂₂sinθ
	// c₁₂ = -a₁₁sinθ + a₁₂cosθ
	// c₂₂ = -a₂₁sinθ + a₂₂cosθ
	// c₁₃ = a₁₃
	// c₂₃ = a₂₃
	return Matrix{
		m[0]*cosθ + m[2]*sinθ,
		m[1]*cosθ + m[3]*sinθ,
		-m[0]*sinθ + m[2]*cosθ,
		-m[1]*sinθ + m[3]*cosθ,
		m[4],
		m[5],
	}
}

func (m Matrix) Scale(sx, sy float64) Matrix {
	// b:
	//  sx   0  0    m₁₁ m₁₂ m₁₃
	//   0  sy  0    m₂₁ m₂₂ m₂₃
	//   0   0  1    m₃₁ m₃₂ m₃₃

	// c₁₁ = a₁₁sx
	// c₂₁ = a₂₁sx
	// c₁₂ = a₁₂sy
	// c₂₂ = a₂₂sy
	// c₁₃ = a₁₃
	// c₂₃ = a₂₃
	return Matrix{
		m[0] * sx,
		m[1] * sx,
		m[2] * sy,
		m[3] * sy,
		m[4],
		m[5],
	}
}

// SvgTransformMatrix creates a Matrix from an SVG transform attribute.
func SvgTransformMatrix(svgtransform string) (Matrix, error) {
	k := svgtrtok{p: svgtransform}
	result := MatrixIdentity
	for k.scan() {
		narg := len(k.arg)
		switch k.fn {

		case "matrix":
			if narg != 6 {
				return result, fmt.Errorf("matrix must have 6 arguments, got %d", narg)
			}
			var mat Matrix
			for i := range mat[:] {
				mat[i] = k.arg[i]
			}
			result = result.Mul(mat)

		case "translate":
			if narg != 1 && narg != 2 {
				return result, fmt.Errorf("translate must have 1 or 2 arguments, got %d", narg)
			}
			var cx, cy float64
			k.unpackarg(&cx, &cy)
			result = result.Translate(cx, cy)

		case "rotate":
			if narg != 1 && narg != 3 {
				return result, fmt.Errorf("rotate must have 1 or 3 arguments, got %d", narg)
			}
			var θ, cx, cy float64
			k.unpackarg(&θ, &cx, &cy)
			result = result.Translate(cx, cy).Rotate(θ).Translate(-cx, -cy)

		case "scale":
			if narg != 1 && narg != 2 {
				return result, fmt.Errorf("scale must have 1 or 2 arguments, got %d", narg)
			}
			var sx, sy float64
			k.unpackarg(&sx, &sy)
			if narg == 1 {
				sy = sx
			}
			result.Scale(sx, sy)

		default:
			return result, fmt.Errorf("unknown function %q", k.fn)
		}
	}

	return result, k.err
}

type svgtrtok struct {
	p string
	i int

	fn  string
	arg []float64

	err error
}

func (k *svgtrtok) scan() bool {
	if k.err != nil {
		return false
	}

	for k.ch() == ' ' {
		k.i++
	}

	if k.ch() == -1 {
		return false // trailing space
	}

	if !k.parsefn() {
		return false
	}

	k.arg = k.arg[:0]

	for k.parsearg() {
	}

	return k.err == nil
}

func (k *svgtrtok) unpackarg(argp ...*float64) {
	n := len(argp)
	if m := len(k.arg); m < n {
		n = m
	}
	for i, p := range argp[:n] {
		*p = k.arg[i]
	}
}

func (k *svgtrtok) ch() int {
	if k.i < len(k.p) {
		return int(k.p[k.i])
	}
	return -1
}

func (k *svgtrtok) chstr() string {
	c := k.ch()
	if c == -1 {
		return "EOF"
	}

	return fmt.Sprintf("%q", c)
}

func (k *svgtrtok) parsefn() bool {
	start := k.i

	isalpha := func() bool {
		c := k.ch()
		return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
	}

	for isalpha() {
		k.i++
	}

	if start == k.i {
		k.err = fmt.Errorf("expected function name at %v, got %v", start, k.chstr())
		return false
	}

	k.fn = k.p[start:k.i]

	for k.ch() == ' ' {
		k.i++
	}

	if k.ch() != '(' {
		k.err = fmt.Errorf("expected '(' after function name %q at %v, got %v", k.fn, start, k.chstr())
		return false
	}

	k.i++ // skip past '('

	return true
}

func (k *svgtrtok) parsearg() bool {
	for k.ch() == ' ' {
		k.i++
	}

	start := k.i

	maybenum := func() bool {
		c := k.ch()
		return c != ' ' && c != ',' && c != ')'
	}

	for maybenum() {
		k.i++
	}

	if start == k.i {
		k.err = fmt.Errorf("expected function argument at %v, got %v", start, k.chstr())
		return false
	}

	sarg := k.p[start:k.i]
	arg, err := strconv.ParseFloat(sarg, 64)
	if err != nil {
		k.err = fmt.Errorf("expected function argument at %v, got %q", start, sarg)
		return false
	}

	k.arg = append(k.arg, arg)

	for k.ch() == ' ' {
		k.i++
	}

	c := k.ch()
	if c == ',' {
		k.i++
		return true
	}

	if c == ')' {
		k.i++
		return false
	}

	return true // continue parsing
}
