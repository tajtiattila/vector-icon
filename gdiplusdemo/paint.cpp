
#include "sys.h"

#pragma comment (lib, "Msimg32.lib")

////////////////////////////////////////////////////////////////////////////////

void PaintIcon16x16Clipped(Gdiplus::Graphics& graphics) {
	using namespace Gdiplus;

	// Set pixel offset mode so that the middle of the top left pixel
	// is at (0.5, 0.5). With that integer coordinates represents
	// the pixel grid like in SVG, not pixels themselves.
	graphics.SetPixelOffsetMode(PixelOffsetModeHalf);

	GraphicsPath tri;
	tri.AddLine(7, 0, 14, 14);
	tri.AddLine(14, 14, 0, 14);
	tri.AddLine(0, 14, 7, 0);

	GraphicsPath clipPath;
	clipPath.AddEllipse(7, 7, 10, 10);

	Region clipRegion(&clipPath);
	graphics.ExcludeClip(&clipRegion);

	SolidBrush blue(Color(255, 0, 0, 255));
	graphics.FillPath(&blue, &tri);

	graphics.ResetClip();

	GraphicsPath circle;
	circle.AddEllipse(8, 8, 8, 8);

	SolidBrush red(Color(255, 255, 0, 0));
	graphics.FillPath(&red, &circle);
}

void PaintIcon16x16(Gdiplus::Graphics& graphics) {
	using namespace Gdiplus;

	// Set pixel offset mode so that the middle of the top left pixel
	// is at (0.5, 0.5). With that integer coordinates represents
	// the pixel grid like in SVG, not pixels themselves.
	graphics.SetPixelOffsetMode(PixelOffsetModeHalf);

// d="M 7,0 0,14 H 7.4277344 C 7.1547433,13.376385 7.0002047,12.699 7,12 c -2e-7,-2.2752659 1.5296125,-4.173664 3.609375,-4.7792969 z"

	GraphicsPath tri;
	tri.AddLine(7.f, 0.f, 0.f, 14.f);
	tri.AddLine(0.f, 14.f, 7.4277344f, 14.f);
	tri.AddBezier(7.4277344f, 14.f, 7.1547433f, 13.376385f, 7.f, 12.699f, 7.f, 12.f);
	tri.AddBezier(7.f, 12.f, 7.f, 9.7247341f, 8.5296125f, 7.826336f, 10.609375f, 7.2207031f);
	tri.AddLine(10.609375f, 7.2207031f, 7.f, 0.f);

	SolidBrush blue(Color(255, 0, 0, 255));
	graphics.FillPath(&blue, &tri);

	GraphicsPath circle;
	circle.AddEllipse(8, 8, 8, 8);

	SolidBrush red(Color(255, 255, 0, 0));
	graphics.FillPath(&red, &circle);
}

void PaintGrid(Gdiplus::Graphics& graphics,
		int ox, int oy, int nx, int ny, int step) {

	using namespace Gdiplus;

	PixelOffsetMode pom = graphics.GetPixelOffsetMode();
	graphics.SetPixelOffsetMode(PixelOffsetModeNone);

	Pen pen(Color(32, 0, 0, 0), 1.0f);

	int ey = oy + ny * step;
	for (int x = ox, i = 0; i <= nx; x += step, i++) {
		graphics.DrawLine(&pen, x, oy, x, ey);
	}

	int ex = ox + nx * step;
	for (int y = oy, i = 0; i <= ny; y += step, i++) {
		graphics.DrawLine(&pen, ox, y, ex, y);
	}

	graphics.SetPixelOffsetMode(pom);
}

void PaintGrid(HDC hdc, int ox, int oy, int nx, int ny, int step) {
	static constexpr BYTE gray = 192;
	HPEN pen = ::CreatePen(PS_SOLID, 1, RGB(gray, gray, gray));
	HGDIOBJ oldPen = ::SelectObject(hdc, pen);

	int ex = ox + nx*step;
	int ey = oy + ny*step;
	for (int x = ox, i = 0; i <= nx; x += step, i++) {
		::MoveToEx(hdc, x, oy, nullptr);
		::LineTo(hdc, x, ey);
	}
	for (int y = oy, i = 0; i <= ny; y += step, i++) {
		::MoveToEx(hdc, ox, y, nullptr);
		::LineTo(hdc, ex, y);
	}

	::SelectObject(hdc, oldPen);
	::DeleteObject(pen);
}

void GridPattern(HDC hdc, int dx, int dy) {
	PaintGrid(hdc, 0, 0, dx, dy, 8);
}

Gdiplus::Color AvgPixel(Gdiplus::Bitmap& src, INT sx, INT sy, INT scale) {
	// NOTE(ata): this doesn't work, because
	// Gdiplus::Color uses non-premultiplied color values.
	UINT r = 0;
	UINT g = 0;
	UINT b = 0;
	UINT a = 0;

	Gdiplus::Color c;
	for (INT y = 0; y < scale; y++) {
		for (INT x = 0; x < scale; x++) {
			src.GetPixel(sx+x, sy+y, &c);
			r += c.GetR();
			g += c.GetG();
			b += c.GetB();
			a += c.GetA();
		}
	}

	UINT n = scale*scale;

	r = (r + (n/2)) / n;
	g = (g + (n/2)) / n;
	b = (b + (n/2)) / n;
	a = (a + (n/2)) / n;

	return Gdiplus::Color(BYTE(a), BYTE(r), BYTE(g), BYTE(b));
}

// AvgPixel averages a pixel assuming alpha-premultiplied colors.
void AvgPixel(BYTE* pdest, const BYTE* psrc, INT scale, INT stride) {
	UINT a = 0;
	UINT r = 0;
	UINT g = 0;
	UINT b = 0;

	const BYTE* py = psrc;
	for (INT y = 0; y < scale; y++) {
		const BYTE* ppix = py;
		for (INT x = 0; x < scale; x++) {
			a += *ppix++;
			r += *ppix++;
			g += *ppix++;
			b += *ppix++;
		}
		py += stride;
	}

	INT n = scale*scale;
	*pdest++ = (a + (n/2)) / n;
	*pdest++ = (r + (n/2)) / n;
	*pdest++ = (g + (n/2)) / n;
	*pdest   = (b + (n/2)) / n;
}

void SimpleDownscale(Gdiplus::Bitmap& icon, Gdiplus::Bitmap& src, INT scale) {
	using namespace Gdiplus;

	const INT ix = icon.GetWidth();
	const INT iy = icon.GetHeight();

	Rect irect(0, 0, ix, iy);
	BitmapData ibits;
	icon.LockBits(&irect, ImageLockModeWrite, PixelFormat32bppPARGB, &ibits);

	Rect srect(0, 0, ix*scale, iy*scale);
	BitmapData sbits;
	src.LockBits(&srect, ImageLockModeRead, PixelFormat32bppPARGB, &sbits);

	BYTE* sp = (BYTE*)sbits.Scan0;
	BYTE* ip = (BYTE*)ibits.Scan0;
	for (INT y = 0; y < iy; y++) {
		BYTE* ip = (BYTE*)ibits.Scan0 + y * ibits.Stride;
		BYTE* sp = (BYTE*)sbits.Scan0 + y * scale * sbits.Stride;
		for (INT x = 0; x < ix; x++) {
			AvgPixel(ip, sp, scale, sbits.Stride);
			ip += 4;
			sp += scale * 4;
		}
	}

	src.UnlockBits(&sbits);
	icon.UnlockBits(&ibits);
}

void GdiPlusDownscale(Gdiplus::Bitmap& icon, Gdiplus::Bitmap& src, INT scale) {
	using namespace Gdiplus;

	const INT ix = icon.GetWidth();
	const INT iy = icon.GetHeight();

	Graphics gr(&icon);

	gr.SetInterpolationMode(InterpolationModeHighQuality);
	//gr.SetInterpolationMode(InterpolationModeBilinear);
	//gr.SetInterpolationMode(InterpolationModeHighQualityBilinear);

	gr.SetCompositingQuality(CompositingQualityGammaCorrected);
	gr.SetCompositingMode(CompositingModeSourceCopy);
	gr.SetPixelOffsetMode(PixelOffsetModeHalf);

	gr.DrawImage(&src, REAL(0), REAL(0), REAL(ix), REAL(iy));

	/*
	RectF destRect(0, 0, REAL(ix), REAL(iy));
	RectF sourceRect(0, 0, REAL(ix*scale), REAL(iy*scale));

	gr.DrawImage(&bm, destRect, sourceRect, UnitPixel, nullptr);
	*/
}

void PaintIconDownscaled(Gdiplus::Bitmap& icon, UINT scale) {
	using namespace Gdiplus;

	const UINT ix = icon.GetWidth();
	const UINT iy = icon.GetHeight();

	Bitmap bm(ix*scale, iy*scale, PixelFormat32bppPARGB);
	Graphics scalegr(&bm);

	// should have no effect with scale ~ 8 and above
	//scalegr.SetSmoothingMode(SmoothingModeHighQuality);

	scalegr.ScaleTransform(REAL(scale), REAL(scale), MatrixOrderPrepend);

	PaintIcon16x16(scalegr);

	SimpleDownscale(icon, bm, scale);
	//GdiPlusDownscale(icon, bm, scale);
}

void PaintZoomed(Gdiplus::Graphics& gr,
		Gdiplus::Image& image, int x, int y, int scale) {
	using namespace Gdiplus;

	int ix = image.GetWidth();
	int iy = image.GetHeight();

	gr.SetPixelOffsetMode(PixelOffsetModeHalf);
	gr.SetInterpolationMode(InterpolationModeNearestNeighbor);
	gr.DrawImage(&image, x, y, ix * scale, iy * scale);

	PaintGrid(gr, x, y, ix, iy, scale);
}

void OnPaint(HDC hdc, int dx, int dy) {
	using namespace Gdiplus;

	// Icon size and render scale
	const int ix = 16;
	const int iy = 16;

	// Layout
	const int padding = 16;
	const int xfree = dx - ix - 3*padding;
	const int xone = xfree/3;

	const int yone = dy - padding;
	const int xscalmin = xone < yone ? xone : yone;

	const int zoom = (xscalmin - padding) / ix;
	const int xscal = ix * zoom;

	const int xsrc = padding;
	const int xicon = xsrc + xscal + padding;
	const int xdest1 = xicon + ix + padding;
	const int xdest2 = xdest1 + xscal + padding;

	Graphics gr(hdc);
	gr.SetPixelOffsetMode(PixelOffsetModeHalf);

	// paint source in zoomed canvas
	Bitmap bm(ix*zoom, iy*zoom, PixelFormat32bppPARGB);
	Graphics offgr(&bm);
	offgr.SetSmoothingMode(SmoothingModeHighQuality);
	offgr.ScaleTransform(REAL(zoom), REAL(zoom), MatrixOrderPrepend);

	PaintIcon16x16(offgr);

	// display zoomed canvas and grid
	gr.DrawImage(&bm, xsrc, padding);
	PaintGrid(gr, xsrc, padding, ix, iy, zoom);

	// paint upscaled image and display in normal size
	Bitmap icon(ix, iy, PixelFormat32bppPARGB);
	PaintIconDownscaled(icon, 8);
	gr.DrawImage(&icon, xicon, padding);

	// display downscaled image, zoomed in
	PaintZoomed(gr, icon, xdest1, padding, zoom);

	// paint and display 1x1 scale image
	Bitmap icon2(ix, iy, PixelFormat32bppPARGB);
	Graphics gr2(&icon2);
	gr2.SetSmoothingMode(SmoothingModeAntiAlias8x8);
	PaintIcon16x16(gr2);

	// display 1x1 image, zoomed in
	PaintZoomed(gr, icon2, xdest2, padding, zoom);
	gr.DrawImage(&icon2, xicon, padding + iy + padding);
}
