package main

import (
	"math"
	"testing"
)

func TestSvgTransformMatrix(t *testing.T) {
	// https://www.w3.org/TR/SVG/images/coords/NestedCalcs.png
	want := Matrix{0.707, -0.707, .707, .707, 255.06, 111.21}
	m, err := SvgTransformMatrix("translate(50,90) rotate(-45) translate(130,160)")
	if err != nil {
		t.Fatal("invalid transformation:", err)
	}

	for i := range want[:] {
		if math.Abs(want[i]-m[i]) > 0.005 {
			t.Fatalf("invalid matrix, want %.3f got %.3f", want, m)
		}
	}
}
