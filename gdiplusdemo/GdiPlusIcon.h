#ifndef GDIPLUSICON_H_INCLUDED_
#define GDIPLUSICON_H_INCLUDED_

#include <windows.h>

#define GDIPVER 0x0110 // Windows Vista or later
#include <Gdiplus.h>

#include "IconPack.h"

class GdiPlusIconEngine : public vectoricon::DrawEngine {
public:
	GdiPlusIconEngine();

	void DrawIcon(HDC hdc, RECT const* r, vectoricon::Icon const& icon);

	// vectoricon::DrawEngine overrides
	void ViewBox(float xmin, float ymin, float xmax, float ymax) override;
	void SetSolidFill(uint8_t r, uint8_t g, uint8_t b, uint8_t a) override;
	void MoveTo(vectoricon::Point p) override;
	void LineTo(std::vector<vectoricon::Point> const& p) override;
	void CubicBezierTo(std::vector<vectoricon::Point> const& p) override;
	void QuadraticBezierTo(std::vector<vectoricon::Point> const& p) override;
	void ClosePath() override;

private:
	std::pair<const Gdiplus::PointF*, INT>
		convertPoints(std::vector<vectoricon::Point> const& pts);

private:
	Gdiplus::Bitmap m_bitmap; // bitmap used for painting
	Gdiplus::Graphics m_graphics;

	Gdiplus::SolidBrush m_emptyBrush;
	Gdiplus::SolidBrush m_solidBrush;
	Gdiplus::GraphicsPath m_path;

	vectoricon::Point m_current = {0.f, 0.f};
	std::vector<Gdiplus::PointF> m_ptbuf;

	int m_dx = 0;
	int m_dy = 0;
	bool m_dirty = false;
};

#endif // GDIPLUSICON_H_INCLUDED_
