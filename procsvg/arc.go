package main

import (
	"math"
)

// see https://github.com/colinmeinke/svg-arc-to-cubic-bezier/blob/master/src/index.js

const τ = math.Pi * 2

func arcToBezier(p, c, r Point, xAxisRot float64, largeArc, sweep bool) []Point {
	px, py := p.X, p.Y
	cx, cy := c.X, c.Y
	rx, ry := math.Abs(r.X), math.Abs(r.Y)

	if rx == 0 || ry == 0 {
		return nil
	}

	sinφ := math.Sin(xAxisRot * τ / 360)
	cosφ := math.Cos(xAxisRot * τ / 360)

	pxp := cosφ*(px-cx)/2 + sinφ*(py-cy)/2
	pyp := -sinφ*(px-cx)/2 + cosφ*(py-cy)/2

	if pxp == 0 && pyp == 0 {
		return nil
	}

	λ := sq(pxp)/sq(rx) + sq(pyp)/sq(ry)
	if λ > 1 {
		sqrtλ := math.Sqrt(λ)
		rx *= sqrtλ
		ry *= sqrtλ
	}

	centerx, centery, ang1, ang2 := getArcCenter(px, py, cx, cy, rx, ry,
		largeArc, sweep, sinφ, cosφ, pxp, pyp)

	// If 'ang2' == 90.0000000001, then `ratio` will evaluate to
	// 1.0000000001. This causes `segments` to be greater than one, which is an
	// unecessary split, and adds extra points to the bezier curve. To alleviate
	// this issue, we round to 1.0 when the ratio is close to 1.0.
	ratio := math.Abs(ang2) / (τ / 4)
	if math.Abs(1.0-ratio) < 0.0000001 {
		ratio = 1.0
	}

	nseg := int(math.Ceil(ratio))
	if nseg == 0 {
		nseg = 1
	}

	ang2 /= float64(nseg)

	var pts []Point

	for i := 0; i < nseg; i++ {
		curve := approxUnitArc(ang1, ang2)

		for _, c := range curve {
			pts = append(pts, mapToEllipse(c, rx, ry, cosφ, sinφ, centerx, centery))
		}

		ang1 += ang2
	}

	return pts
}

func getArcCenter(px, py, cx, cy, rx, ry float64, largeArc, sweep bool,
	sinφ, cosφ, pxp, pyp float64) (centerx, centery, ang1, ang2 float64) {

	rxsq := sq(rx)
	rysq := sq(ry)
	pxpsq := sq(pxp)
	pypsq := sq(pyp)

	radicant := (rxsq * rysq) - (rxsq * pypsq) - (rysq * pxpsq)

	if radicant < 0 {
		radicant = 0
	}

	radicant /= (rxsq * pypsq) + (rysq * pxpsq)
	radicant = math.Sqrt(radicant)
	if largeArc == sweep {
		radicant *= -1
	}

	centerxp := radicant * rx / ry * pyp
	centeryp := radicant * -ry / rx * pxp

	centerx = cosφ*centerxp - sinφ*centeryp + (px+cx)/2
	centery = sinφ*centerxp + cosφ*centeryp + (py+cy)/2

	vx1 := (pxp - centerxp) / rx
	vy1 := (pyp - centeryp) / ry
	vx2 := (-pxp - centerxp) / rx
	vy2 := (-pyp - centeryp) / ry

	ang1 = vectorAngle(1, 0, vx1, vy1)
	ang2 = vectorAngle(vx1, vy1, vx2, vy2)

	if !sweep && ang2 > 0 {
		ang2 -= τ
	}

	if sweep && ang2 < 0 {
		ang2 += τ
	}

	return centerx, centery, ang1, ang2
}

func vectorAngle(ux, uy, vx, vy float64) float64 {
	var sign float64
	if ux*vy-uy*vx < 0 {
		sign = -1
	} else {
		sign = 1
	}

	dot := ux*vx + uy*vy

	if dot > 1 {
		dot = 1
	}

	if dot < -1 {
		dot = -1
	}

	return sign * math.Acos(dot)
}

func approxUnitArc(ang1, ang2 float64) []Point {

	// For 90° a circular arc, use a constant as derived from
	// http://spencermortensen.com/articles/bezier-circle
	var a float64
	if ang2 == math.Pi/2 {
		a = 0.551915024494
	} else if ang2 == -math.Pi/2 {
		a = -0.551915024494
	} else {
		a = 4 / 3 * math.Tan(ang2/4)
	}

	x1 := math.Cos(ang1)
	y1 := math.Sin(ang1)
	x2 := math.Cos(ang1 + ang2)
	y2 := math.Sin(ang1 + ang2)

	return []Point{
		{
			X: x1 - y1*a,
			Y: y1 + x1*a,
		},
		{
			X: x2 + y2*a,
			Y: y2 - x2*a,
		},
		{
			X: x2,
			Y: y2,
		},
	}
}

func mapToEllipse(p Point, rx, ry, cosφ, sinφ, centerx, centery float64) Point {
	x, y := p.X, p.Y

	x *= rx
	y *= ry

	xp := cosφ*x - sinφ*y
	yp := sinφ*x + cosφ*y

	return Point{
		X: xp + centerx,
		Y: yp + centery,
	}
}

func sq(v float64) float64 {
	return v * v
}
