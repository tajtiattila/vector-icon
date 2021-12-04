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

void GdiPlusIconEngine::DrawIconDirect(HDC hdc, RECT const* rr, vectoricon::Icon const& icon) {
	using namespace Gdiplus;

	RECT const& r = *rr;
	m_ox = r.left;
	m_oy = r.top;
	m_dx = r.right - r.left;
	m_dy = r.bottom - r.top;

	Graphics gr(hdc);

	gr.SetPixelOffsetMode(PixelOffsetModeHalf);
	gr.SetSmoothingMode(SmoothingModeAntiAlias8x8);
	gr.SetClip(Rect{m_ox, m_oy, m_dx, m_dy}, CombineModeReplace);

	m_gr = &gr;

	DrawIconImpl(icon);

	m_gr = nullptr;
}

void GdiPlusIconEngine::DrawIcon(HDC hdc, RECT const* rr, vectoricon::Icon const& icon) {
	RECT const& r = *rr;
	m_ox = 0;
	m_oy = 0;
	m_dx = r.right - r.left;
	m_dy = r.bottom - r.top;

	m_gr = &m_graphics;

	if (m_dirty) {
		m_graphics.SetCompositingMode(Gdiplus::CompositingModeSourceCopy);
		m_graphics.SetSmoothingMode(Gdiplus::SmoothingModeNone);

		m_graphics.FillRectangle(&m_emptyBrush, 0, 0, m_dx, m_dy);
	}

	m_graphics.SetSmoothingMode(Gdiplus::SmoothingModeAntiAlias8x8);
	m_graphics.SetCompositingMode(Gdiplus::CompositingModeSourceOver);

	DrawIconImpl(icon);

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

void GdiPlusIconEngine::DrawIconImpl(vectoricon::Icon const& icon) {
	m_currentPathIdx = 1;

	vectoricon::DrawIcon(icon, m_dx, m_dy, this);
}

void GdiPlusIconEngine::DebugSinglePath(size_t n) {
	m_debugPathIdx = n;
}

void GdiPlusIconEngine::ViewBox(float xmin, float ymin, float xmax, float ymax) {
	float vx = xmax - xmin;
	float vy = ymax - ymin;
	float xscale = float(m_dx)/vx;
	float yscale = float(m_dy)/vy;
	Gdiplus::Matrix m(xscale, 0.f, 0.f, yscale, m_ox-xmin, m_oy-ymin);
	m_gr->SetTransform(&m);
}

void GdiPlusIconEngine::SetSolidFill(uint8_t r, uint8_t g, uint8_t b, uint8_t a) {
	m_solidBrush.SetColor(Gdiplus::Color(a, r, g, b));
}

void GdiPlusIconEngine::MoveTo(vectoricon::Point p) {
	endPath();

	m_cursor = p;
	m_startp = p;
}

void GdiPlusIconEngine::LineTo(std::vector<vectoricon::Point> const& p) {
	auto [pts, n] = convertPoints(p);
	m_path.AddLines(pts, n);

	m_hasPath = true;
}

void GdiPlusIconEngine::CubicBezierTo(std::vector<vectoricon::Point> const& p) {
	auto [pts, n] = convertPoints(p);
	m_path.AddBeziers(pts, n);

	m_hasPath = true;
}

void GdiPlusIconEngine::QuadraticBezierTo(std::vector<vectoricon::Point> const& pts) {
	size_t nsegments = pts.size() / 2;

	m_ptbuf.clear();
	m_ptbuf.reserve(1+3*nsegments);

	vectoricon::Point q0 = m_cursor;

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

	m_cursor = q0;

	m_path.AddBeziers(m_ptbuf.data(), (INT)m_ptbuf.size());

	m_hasPath = true;
}

void GdiPlusIconEngine::ClosePath() {
	endPath();

	if (m_hasPath) {
		if (m_debugPathIdx == 0 || m_currentPathIdx == m_debugPathIdx) {
			m_gr->FillPath(&m_solidBrush, &m_path);
		}

		m_currentPathIdx++;
	}

	m_path.Reset();

	m_hasPath = false;
}

void GdiPlusIconEngine::endPath() {
	if (m_hasPath && (m_cursor.x != m_startp.x || m_cursor.y != m_startp.y)) {
		m_path.AddLine(m_cursor.x, m_cursor.y, m_startp.x, m_startp.y);
	}
}

std::pair<const Gdiplus::PointF*, INT>
GdiPlusIconEngine::convertPoints(std::vector<vectoricon::Point> const& pts) {
	m_ptbuf.clear();
	m_ptbuf.reserve(1+pts.size());

	m_ptbuf.push_back({m_cursor.x, m_cursor.y});
	for (auto const& p : pts) {
		m_ptbuf.push_back({p.x, p.y});
	}

	m_cursor = pts.back();

	return {m_ptbuf.data(), (INT)m_ptbuf.size()};
}
