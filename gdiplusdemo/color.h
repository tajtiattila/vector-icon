#ifndef COLORSPACE_H_INCLUDED_
#define COLORSPACE_H_INCLUDED_

#include <cstdint>
#include <cmath>

namespace colorspace {

struct sRGB;
struct XYZ;
struct Lab;

struct sRGB {
	uint8_t r;
	uint8_t g;
	uint8_t b;

	static sRGB from(XYZ color);
	static sRGB from(Lab color);
};

// CIE XYZ color. X, Y and Z refer to a D65/2° standard illuminant.
struct XYZ {
	double X, Y, Z;

	static XYZ from(sRGB color);
	static XYZ from(Lab color);
};

// CIE-L*ab color.
struct Lab {
	double L, a, b;

	static constexpr double RefX = 95.047;
	static constexpr double RefY = 100.000;
	static constexpr double RefZ = 108.883;

	static Lab from(XYZ color);
	static Lab from(sRGB color);
};

inline XYZ XYZ::from(sRGB s) {
	//sR, sG and sB (Standard RGB) input range = 0 ÷ 255
	//X, Y and Z output refer to a D65/2° standard illuminant.

	auto f = [](uint8_t s) -> double {
		double v = double(s) / 255;
		if (v > 0.04045) {
			return std::pow((v+0.055) / 1.055, 2.4);
		}
		return v / 12.92;
	};

	double r = f(s.r)*100;
	double g = f(s.g)*100;
	double b = f(s.b)*100;

	XYZ c;
	c.X = r*0.41239079926595934 + g*0.357584339383878   + b*0.1804807884018343;
	c.Y = r*0.21263900587151027 + g*0.715168678767756   + b*0.07219231536073371;
	c.Z = r*0.01933081871559182 + g*0.11919477979462598 + b*0.9505321522496607;
	return c;
}

inline sRGB sRGB::from(XYZ c) {
	//X, Y and Z input refer to a D65/2° standard illuminant.
	//sR, sG and sB (standard RGB) output range = 0 ÷ 255

	double x = c.X / 100;
	double y = c.Y / 100;
	double z = c.Z / 100;

	double r = x* 3.2409699419045226  + y*-1.537383177570094   + z*-0.4986107602930034;
	double g = x*-0.9692436362808796  + y* 1.8759675015077202  + z* 0.04155505740717559;
	double b = x* 0.05563007969699366 + y*-0.20397695888897652 + z* 1.0569715142428786;

	auto f = [](double x) -> uint8_t {
		if (x > 0.0031308) {
			x = 1.055 * std::pow(x, 1.0/2.4) - 0.055;
		} else {
			x *= 12.92;
		}
		int i = int((x*255) + 0.5);
		if (i < 0) {
			return 0;
		}
		if (i > 255) {
			return 255;
		}
		return uint8_t(i);
	};

	sRGB s;
	s.r = f(r);
	s.g = f(g);
	s.b = f(b);
	return s;
}

inline Lab Lab::from(XYZ c) {

	auto f = [](double x) -> double {
		double r = std::pow(x, 1.0/3.0);
		return r > 0.008856 ? r : 7.787*x + 16.0/116.0;
	};

	double x = f(c.X/Lab::RefX);
	double y = f(c.Y/Lab::RefY);
	double z = f(c.Z/Lab::RefZ);

	Lab l;
	l.L = (std::max)(0.0, 116.0*y - 16.0);
	l.a = 500.0*(x - y);
	l.b = 200.0*(y - z);
	return l;
}

inline XYZ XYZ::from(Lab l) {
	double y = (l.L+16.0) / 116.0;
	double x = l.a/500.0 + y;
	double z = y - l.b/200.0;

	auto f = [](double v) -> double {
		v = std::pow(v, 3.0);
		if (v > 3) {
			return v;
		}
		return (v - 16.0/116.0) / 7.787;
	};

	XYZ c;
	c.Y = f(y) * Lab::RefY;
	c.X = f(x) * Lab::RefX;
	c.Z = f(z) * Lab::RefZ;
	return c;
}

inline sRGB sRGB::from(Lab color) {
	XYZ c = XYZ::from(color);
	return from(c);
}

inline Lab Lab::from(sRGB color) {
	XYZ c = XYZ::from(color);
	return from(c);
}

} // end namespace colorspace

#endif // COLORSPACE_H_INCLUDED_
