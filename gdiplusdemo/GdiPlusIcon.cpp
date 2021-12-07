#include "GdiPlusIcon.h"

#pragma comment (lib, "Gdiplus.lib")
#pragma comment (lib, "Msimg32.lib") // AlphaBlend

// DIBBuf is an interoperability buffer between GDI+ and GDI.
//
// Using DIBBuf to paint using GDI+ in an HDC can be faster
// because copying the destination content of the GDI to and from GDI+ bitmap
// in the Graphics constructor is be avoided.
class GdiPlusIconEngine::DIBBuf {
public:
	DIBBuf(int dx, int dy) :
		dx(dx),
		dy(dy),
		xbitmap_(dx, dy, PixelFormat32bppPARGB),
		graphics_(&xbitmap_) {

		graphics_.SetPixelOffsetMode(Gdiplus::PixelOffsetModeHalf);
		graphics_.SetSmoothingMode(Gdiplus::SmoothingModeAntiAlias8x8);

		BITMAPINFO bmi;
		ZeroMemory(&bmi, sizeof(bmi));

		BITMAPINFOHEADER& h = bmi.bmiHeader;
		h.biSize = sizeof(BITMAPINFOHEADER);
		h.biWidth = dx;
		h.biHeight = dy;
		h.biPlanes = 1;
		h.biBitCount = 32;
		h.biCompression = BI_RGB;

		hbitmap_ = ::CreateDIBSection(nullptr,
				&bmi, DIB_RGB_COLORS, &bits_, nullptr, 0);
	}

	~DIBBuf() {
		::DeleteObject(hbitmap_);
	}

	// CopyBits copies image data from the GDI+ Bitmap
	// to the GDI DIBSection bitmap.
	bool CopyBits() {
		Gdiplus::Rect rc(0, 0, dx, dy);

		Gdiplus::BitmapData data;
		if (Gdiplus::Ok != xbitmap_.LockBits(&rc, Gdiplus::ImageLockModeRead,
			PixelFormat32bppPARGB, &data)) {
			return false;
		}

		uint8_t* p0 = ((uint8_t*)data.Scan0) + (dy-1)*data.Stride;
		uint8_t* p1 = (uint8_t*)bits_;
		size_t bytesPerLine = 4 * dx;
		for (int y = 0; y < dy; y++) {
			memcpy(p1, p0, bytesPerLine);
			p0 -= data.Stride;
			p1 += bytesPerLine;
		}

		xbitmap_.UnlockBits(&data);
		return true;
	}

	// DrawImage draws the GDI DIBSection bitmap on the HDC.
	void DrawImage(HDC hdc, int x, int y) {
		BLENDFUNCTION bf = {};
		bf.BlendOp = AC_SRC_OVER;
		bf.BlendFlags = 0;
		bf.SourceConstantAlpha = 255;
		bf.AlphaFormat = AC_SRC_ALPHA;

		HDC hdcSrc = ::CreateCompatibleDC(hdc);
		HGDIOBJ hold = ::SelectObject(hdcSrc, hbitmap_);
		::AlphaBlend(hdc, x, y, dx, dy,
			hdcSrc, 0, 0, dx, dy, bf);
		::SelectObject(hdcSrc, hold);
		::DeleteObject(hdcSrc);
	}

	Gdiplus::Graphics& Graphics() { return graphics_; }

	int Dx() const { return dx; }
	int Dy() const { return dy; }

private:
	int dx, dy;
	Gdiplus::Bitmap xbitmap_;
	Gdiplus::Graphics graphics_;

	HBITMAP hbitmap_;
	void* bits_;
};

////////////////////////////////////////////////////////////////////////////////

GdiPlusIconEngine::GdiPlusIconEngine() :
	m_solidBrush(Gdiplus::Color(0, 0, 0, 0)) {
}

void GdiPlusIconEngine::DrawIconEx(bool direct, HDC hdc, RECT const* rr,
		vectoricon::Icon const& icon, size_t palidx) {
	if (direct) {
		DrawIconDirect(hdc, rr, icon, palidx);
	} else {
		DrawIcon(hdc, rr, icon, palidx);
	}
}

// DrawIconDirect draws the icon directly on the destination HDC.
void GdiPlusIconEngine::DrawIconDirect(HDC hdc, RECT const* rr,
		vectoricon::Icon const& icon, size_t palidx) {
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

	DrawIconImpl(icon, palidx);

	m_gr = nullptr;
}

// DrawIcon draws the icon using an internal buffer, eliminating
// the overhead of initializing graphics objects.
void GdiPlusIconEngine::DrawIcon(HDC hdc, RECT const* rr,
		vectoricon::Icon const& icon, size_t palidx) {
	RECT const& r = *rr;
	m_ox = 0;
	m_oy = 0;
	m_dx = r.right - r.left;
	m_dy = r.bottom - r.top;

	uint32_t sz = (uint32_t(m_dx)<<16) | uint32_t(m_dy);
	auto it = m_dibs.find(sz);
	if (it == m_dibs.end()) {
		it = m_dibs.insert({sz, std::make_shared<DIBBuf>(m_dx, m_dy)}).first;
	}

	DIBBuf& buf = *it->second;
	m_gr = &buf.Graphics();

	m_gr->Clear(Gdiplus::Color(0, 0, 0, 0));
	m_gr->ResetTransform();

	DrawIconImpl(icon, palidx);

	buf.CopyBits();
	buf.DrawImage(hdc, r.left, r.top);
}

void GdiPlusIconEngine::DrawIconImpl(vectoricon::Icon const& icon, size_t palidx) {
	m_currentPathIdx = 1;

	icon.Draw(this, m_dx, m_dy, palidx);
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

void GdiPlusIconEngine::Colorize(uint8_t& /*r*/, uint8_t& /*g*/, uint8_t& /*b*/) {
}

void GdiPlusIconEngine::SetSolidFill(uint8_t r, uint8_t g, uint8_t b, uint8_t a) {
	Colorize(r, g, b);
	m_solidBrush.SetColor(Gdiplus::Color(a, r, g, b));
}

void GdiPlusIconEngine::MoveTo(vectoricon::Point p) {
	if (m_hasPath) {
		m_path.CloseFigure();
	}
	m_path.StartFigure();

	m_cursor = p;
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
	m_path.CloseFigure();

	if (m_hasPath) {
		if (m_currentPathIdx == m_debugPathIdx) {
			Gdiplus::PathData pd;
			m_path.GetPathData(&pd);
		}

		if (m_debugPathIdx == 0 || m_currentPathIdx == m_debugPathIdx) {
			m_gr->FillPath(&m_solidBrush, &m_path);
		}

		m_currentPathIdx++;
	}

	m_path.Reset();

	m_hasPath = false;
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
