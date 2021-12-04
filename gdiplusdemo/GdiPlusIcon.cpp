#include "GdiPlusIcon.h"

#pragma comment (lib, "Gdiplus.lib")
#pragma comment (lib, "Msimg32.lib") // AlphaBlend

GdiPlusIconEngine::GdiPlusIconEngine() :
	m_bitmap(256, 256, PixelFormat32bppPARGB),
	m_graphics(&m_bitmap),
	m_emptyBrush(Gdiplus::Color(0, 0, 0, 0)),
	m_solidBrush(Gdiplus::Color(0, 0, 0, 0)) {

	m_graphics.SetPixelOffsetMode(Gdiplus::PixelOffsetModeHalf);
}

void GdiPlusIconEngine::DrawIcon(HDC hdc, RECT const* rr, vectoricon::Icon const& icon) {
	RECT const& r = *rr;
	m_dx = r.right - r.left;
	m_dy = r.bottom - r.top;

	if (m_dirty) {
		m_graphics.SetCompositingMode(Gdiplus::CompositingModeSourceCopy);
		m_graphics.SetSmoothingMode(Gdiplus::SmoothingModeNone);

		m_graphics.FillRectangle(&m_emptyBrush, 0, 0, m_dx, m_dy);
	}

	m_graphics.SetSmoothingMode(Gdiplus::SmoothingModeAntiAlias8x8);
	m_graphics.SetCompositingMode(Gdiplus::CompositingModeSourceOver);

	vectoricon::DrawIcon(icon, m_dx, m_dy, this);

	m_dirty = true;

	m_graphics.ResetTransform();

	BLENDFUNCTION bf = {};
	bf.BlendOp = AC_SRC_OVER;
	bf.BlendFlags = 0;
	bf.SourceConstantAlpha = 255;
	bf.AlphaFormat = AC_SRC_ALPHA;

	HDC hdcgr = m_graphics.GetHDC();

	::AlphaBlend(hdc, r.left, r.top, m_dx, m_dy,
			hdcgr, 0, 0, m_dx, m_dy, bf);

	m_graphics.ReleaseHDC(hdcgr);
}

void GdiPlusIconEngine::ViewBox(float xmin, float ymin, float xmax, float ymax) {
	float vx = xmax - xmin;
	float vy = ymax - ymin;
	float xscale = float(m_dx)/vx;
	float yscale = float(m_dy)/vy;
	Gdiplus::Matrix m(xscale, 0.f, 0.f, yscale, -xmin, -ymin);
	m_graphics.SetTransform(&m);
}

void GdiPlusIconEngine::SetSolidFill(uint8_t r, uint8_t g, uint8_t b, uint8_t a) {
	m_solidBrush.SetColor(Gdiplus::Color(a, r, g, b));
}

void GdiPlusIconEngine::MoveTo(vectoricon::Point p) {
	m_current = p;
}

void GdiPlusIconEngine::LineTo(std::vector<vectoricon::Point> const& p) {
	auto [pts, n] = convertPoints(p);
	m_path.AddLines(pts, n);
}

void GdiPlusIconEngine::CubicBezierTo(std::vector<vectoricon::Point> const& p) {
	auto [pts, n] = convertPoints(p);
	m_path.AddBeziers(pts, n);
}

void GdiPlusIconEngine::QuadraticBezierTo(std::vector<vectoricon::Point> const& pts) {
	size_t nsegments = pts.size() / 2;

	m_ptbuf.clear();
	m_ptbuf.reserve(1+3*nsegments);

	vectoricon::Point q0 = m_current;

	m_ptbuf.push_back({q0.x, q0.y});

	// convert quadratics to cubic Béziers
	for (size_t i = 0; i < pts.size(); i +=2 ) {
		vectoricon::Point q1 = pts[i];
		vectoricon::Point q2 = pts[i+1];

		vectoricon::Point c1{
			q0.x + (2.f/3.f)*(q1.x - q0.x),
			q0.y + (2.f/3.f)*(q1.y - q0.y),
		};
		vectoricon::Point c2{
			q2.x + (2.f/3.f)*(q1.x - q2.x),
			q2.y + (2.f/3.f)*(q1.y - q2.y),
		};
		vectoricon::Point c3 = q2;

		m_ptbuf.push_back({c1.x, c1.y});
		m_ptbuf.push_back({c2.x, c2.y});
		m_ptbuf.push_back({c3.x, c3.y});

		q0 = q2;
	}

	m_current = q0;

	m_path.AddBeziers(m_ptbuf.data(), (INT)m_ptbuf.size());
}

void GdiPlusIconEngine::ClosePath() {
	m_graphics.FillPath(&m_solidBrush, &m_path);
	m_path.Reset();
}

std::pair<const Gdiplus::PointF*, INT>
GdiPlusIconEngine::convertPoints(std::vector<vectoricon::Point> const& pts) {
	m_ptbuf.clear();
	m_ptbuf.reserve(1+pts.size());

	m_ptbuf.push_back({m_current.x, m_current.y});
	for (auto const& p : pts) {
		m_ptbuf.push_back({p.x, p.y});
	}

	m_current = pts.back();

	return {m_ptbuf.data(), (INT)m_ptbuf.size()};
}
