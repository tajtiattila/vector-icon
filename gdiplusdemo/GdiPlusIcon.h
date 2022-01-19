#ifndef GDIPLUSICON_H_INCLUDED_
#define GDIPLUSICON_H_INCLUDED_

#include <windows.h>

#include <objidl.h>

#define GDIPVER 0x0110 // Windows Vista or later
#include <Gdiplus.h>

#include "IconPack.h"

class GdiPlusIconEngine : public vectoricon::DrawEngine {
public:
	GdiPlusIconEngine();

	size_t GetPaletteIndex() const { return m_palIdx; }
	void SetPaletteIndex(size_t paletteIndex) { m_palIdx = paletteIndex; }

	void SetColorization(vectoricon::ColorOverride* co) { m_colorOverride = co; }

	void DrawIcon(HDC hdc, RECT const* r, vectoricon::Icon const& icon);
	void DrawIconDirect(HDC hdc, RECT const* r, vectoricon::Icon const& icon);
	void DrawIconEx(bool direct, HDC hdc, RECT const* r, vectoricon::Icon const& icon);

	HICON CreateBitmapIcon(SIZE size, vectoricon::Icon const& icon);

	virtual void Colorize(uint8_t& r, uint8_t& g, uint8_t& b);

	// vectoricon::DrawEngine overrides
	void ViewBox(float xmin, float ymin, float xmax, float ymax) override;
	void SetSolidFill(uint8_t r, uint8_t g, uint8_t b, uint8_t a) override;
	void MoveTo(vectoricon::Point p) override;
	void LineTo(std::vector<vectoricon::Point> const& p) override;
	void CubicBezierTo(std::vector<vectoricon::Point> const& p) override;
	void QuadraticBezierTo(std::vector<vectoricon::Point> const& p) override;
	void ClosePath() override;

	void DebugSinglePath(size_t n);

private:
	class DIBBuf;

	DIBBuf& DrawIconBuf(SIZE size, vectoricon::Icon const& icon);
	void DrawIconImpl(vectoricon::Icon const& icon);

	std::pair<const Gdiplus::PointF*, INT>
		convertPoints(std::vector<vectoricon::Point> const& pts);

private:
	std::unordered_map<uint32_t, std::shared_ptr<DIBBuf>> m_dibs;

	Gdiplus::Graphics* m_gr;

	Gdiplus::SolidBrush m_solidBrush;
	Gdiplus::GraphicsPath m_path;

	vectoricon::Point m_cursor = {0.f, 0.f};
	bool m_hasPath = false;
	std::vector<Gdiplus::PointF> m_ptbuf;

	size_t m_palIdx = 0;
	vectoricon::ColorOverride* m_colorOverride = nullptr;

	size_t m_currentPathIdx = 0;
	size_t m_debugPathIdx = 0;

	int m_ox = 0;
	int m_oy = 0;
	int m_dx = 0;
	int m_dy = 0;
};

#endif // GDIPLUSICON_H_INCLUDED_
