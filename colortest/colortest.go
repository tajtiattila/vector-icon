package main

import (
	"fmt"
	"image/color"
)

func main() {
	for _, p := range colorpairs {
		l, d := p.light, p.dark
		fmt.Printf("%s %s %.5f %.5f %.5f\n",
			cs(l), cs(d), y(l), y(d), 1-y(l))
	}
}

func cs(c color.Color) string {
	v := color.NRGBAModel.Convert(c).(color.NRGBA)
	return fmt.Sprintf("#%02x%02x%02x", v.R, v.G, v.B)
}

func y(c color.Color) float64 {
	v := color.YCbCrModel.Convert(c).(color.YCbCr)
	return float64(v.Y) / 255
}

type colorpair struct {
	light, dark color.NRGBA
}

var colorpairs []colorpair

func init() {
	f := func(l, d uint32) colorpair {
		lr := uint8(l >> 16)
		lg := uint8(l >> 8)
		lb := uint8(l)
		dr := uint8(d >> 16)
		dg := uint8(d >> 8)
		db := uint8(d)
		return colorpair{
			light: color.NRGBA{lr, lg, lb, 255},
			dark:  color.NRGBA{dr, dg, db, 255},
		}
	}

	colorpairs = []colorpair{
		f(0xEB3D3B, 0xF33636),
		f(0x383838, 0xD4D4D4),
		f(0x2389C9, 0x1F9DDD),
		f(0xE27328, 0xF7CE87),
		f(0xF5D88C, 0xB79701),
		f(0x78797A, 0x9D9D9D),
		f(0xFDFDFD, 0x4C4C4C),
		f(0x91C9EC, 0x096BBB),
		f(0x17A94F, 0x1DCE5C),
		f(0xD5D5D5, 0x636363),
		f(0x21B2C6, 0x20B1C5),
		f(0x9E509D, 0xDEA7D3),
		f(0xDDBBD7, 0xA03D95),
		f(0xABD6B0, 0x2D8E49),
		f(0xBDBDBD, 0x757575),
	}
}
