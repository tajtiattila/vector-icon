#ifndef UNICODE
#define UNICODE
#endif

#define WIN32_LEAN_AND_MEAN
#include <windows.h>

#include "color.h"
#include "GdiPlusIcon.h"

#include <algorithm>
#include <fstream>

class ColorizerIconEngine : public GdiPlusIconEngine {
public:
	void Colorize(uint8_t &r, uint8_t &g, uint8_t &b) override;

	static constexpr COLORREF lightbk = RGB(192, 192, 192);
	static constexpr COLORREF darkbk = RGB(64, 64, 64);

	bool darkMode = false;
	bool grayMode = false;
	int colorMode = 0;
};

class Window {
public:
	void Invalidate();

	void OnKeyDown(WPARAM);
	void OnPaint(HDC dc, int dx, int dy);

	HWND hwnd;
	ColorizerIconEngine* eng;
	vectoricon::Pack pack;

	size_t paintSizeIdx = 0;
	static std::vector<int> paintSizes;

	int debugPathIdx = 0;
	int singleIconIdx = 0;
	bool brushPattern = false;
};

std::vector<int> Window::paintSizes = {16, 20, 24, 28, 32, 48, 64, 128};

Window g_window;

LRESULT CALLBACK WindowProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

int WINAPI wWinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance,
		PWSTR pCmdLine, int nCmdShow) {

	{
		std::ifstream strm("C:/src/rts/head/CLIENT/SRC/RES/rts.iconpk",
				std::ifstream::in | std::ifstream::binary);
		if (!g_window.pack.load(strm)) {
			return 0;
		}
	}

	using namespace Gdiplus;

    // Register the window class.
    const wchar_t CLASS_NAME[] = L"MainWindowClass";

    WNDCLASS wc = {};

    wc.lpfnWndProc    = WindowProc;
    wc.hInstance      = hInstance;
    wc.lpszClassName  = CLASS_NAME;
	wc.hbrBackground  = (HBRUSH)GetStockObject(WHITE_BRUSH);
	wc.hCursor        = LoadCursor(nullptr, IDC_ARROW);

    RegisterClass(&wc);

    // Create the window.

    HWND hwnd = CreateWindowEx(
        0,                              // Optional window styles.
        CLASS_NAME,                     // Window class
        L"GDI Plus",    // Window text
        WS_OVERLAPPEDWINDOW,            // Window style

        // Size and position
        CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT,

        nullptr,    // Parent window
        nullptr,    // Menu
        hInstance,  // Instance handle
        nullptr     // Additional application data
        );

    if (hwnd == nullptr) {
        return 0;
    }

	// Initialize GDI+.
	GdiplusStartupInput gdiplusStartupInput;
	ULONG_PTR           gdiplusToken;
	GdiplusStartup(&gdiplusToken, &gdiplusStartupInput, nullptr);

	g_window.hwnd = hwnd;
	g_window.eng = new ColorizerIconEngine;

    ShowWindow(hwnd, nCmdShow);
	UpdateWindow(hwnd);

    // Run the message loop.

    MSG msg = {};
    while (GetMessage(&msg, NULL, 0, 0)) {
        TranslateMessage(&msg);
        DispatchMessage(&msg);
    }

	GdiplusShutdown(gdiplusToken);

	return 0;
}

LRESULT CALLBACK WindowProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
	switch (msg) {

    case WM_DESTROY:
        PostQuitMessage(0);
        return 0;

    case WM_KEYDOWN:
		if (wParam == VK_ESCAPE) {
			PostQuitMessage(0);
			return 0;
		}
		g_window.OnKeyDown(wParam);
		return 0;

    case WM_SIZE:
		InvalidateRect(hwnd, nullptr, TRUE);
		return 0;

    case WM_PAINT: {
		PAINTSTRUCT ps;
		HDC hdc = BeginPaint(hwnd, &ps);

		RECT rc;
		GetClientRect(hwnd, &rc);
		g_window.OnPaint(hdc, rc.right, rc.bottom);

		EndPaint(hwnd, &ps);
        return 0;
	}

    }

    return DefWindowProc(hwnd, msg, wParam, lParam);
}

void Window::OnKeyDown(WPARAM w) {
	switch (w) {

	case 'B':
		brushPattern = !brushPattern;
		break;

	case 'I':
		paintSizeIdx--;
		if (paintSizeIdx >= paintSizes.size()) {
			paintSizeIdx = paintSizes.size() - 1;
		}
		break;

	case 'K':
		paintSizeIdx++;
		if (paintSizeIdx >= paintSizes.size()) {
			paintSizeIdx = 0;
		}
		break;

	case 'W':
		if (debugPathIdx != 0) {
			debugPathIdx--;
		}
		break;

	case 'S':
		debugPathIdx++;
		break;

	case 'A':
		singleIconIdx--;
		if (singleIconIdx < 0) {
			singleIconIdx = int(pack.size());
		}
		break;

	case 'D':
		singleIconIdx++;
		if (singleIconIdx > pack.size()) {
			singleIconIdx = 0;
		}
		break;

	case 'R':
		eng->colorMode = (eng->colorMode + 1) % 3;
		break;

	case 'Q':
		eng->darkMode = !eng->darkMode;
		break;

	case 'E':
		eng->grayMode = !eng->grayMode;
		break;
	}

	Invalidate();
}

inline uint8_t truncb(int n) {
	if (n < 0) return 0;
	if (255 < n) return 255;
	return (uint8_t)n;
}

uint8_t gray(uint8_t r, uint8_t g, uint8_t b) {
	int ir = r;
	int ig = g;
	int ib = b;
	//  Y' =   0 + (0.299    * R') + (0.587    * G') + (0.114    * B')
	return truncb((ir * 19595 + ig * 38470 + ib * 7471) >> 16);
}

void ToHSL(float& h, float& s, float& l, uint8_t r0, uint8_t g0, uint8_t b0) {
	float r = float(r0) / 255.f;
	float g = float(g0) / 255.f;
	float b = float(b0) / 255.f;

	uint8_t cmax = (std::max)({r0, g0, b0});
	uint8_t cmin = (std::min)({r0, g0, b0});
	float delta = float(cmax - cmin) / 255.f;

	if (cmax == cmin) {
		h = 0;
	} else if (cmax == r0) {
		h = (g-b)/delta;
		if (h < 0) {
			h += 6.f;
		}
	} else if (cmax == g0) {
		h = (b-r)/delta + 2.f;
	} else { // cmax == b0
		h = (r-g)/delta + 4.f;
	}

	int l0 = (int(cmin) + int(cmax));
	l = (float(l0)/2) / 255.f;

	if (l0 == 0 || l0 == 2*255) {
		s = 0;
	} else {
		s = delta/(1.0f - std::fabs(2.f*l - 1.f));
	}
}

void FromHSL(uint8_t& r, uint8_t& g, uint8_t& b, float h, float s, float l) {
	float c = (1.f-fabs(2.f*l-1.f))*s;
	float x = c * (1.f-std::fabs(std::fmod(h, 2.f) - 1.f));
	float m = l - c/2.f;
	static constexpr float z = 0.f;

	std::tuple<float, float, float> fc;
	int ih = int(floor(h));
	switch (ih) {
	case 0: fc = {c, x, z}; break;
	case 1: fc = {x, c, z}; break;
	case 2: fc = {z, c, x}; break;
	case 3: fc = {z, x, c}; break;
	case 4: fc = {x, z, c}; break;
	case 5: fc = {c, z, x}; break;
	}

	r = uint8_t(floor((std::get<0>(fc) + m) * 255.f + 0.5f));
	g = uint8_t(floor((std::get<1>(fc) + m) * 255.f + 0.5f));
	b = uint8_t(floor((std::get<2>(fc) + m) * 255.f + 0.5f));
}

inline void ToYCbCr(uint8_t& y, uint8_t& cb, uint8_t& cr, uint8_t r, uint8_t g, uint8_t b) {
	int ir = r;
	int ig = g;
	int ib = b;

	//  Y' =   0 + (0.299    * R') + (0.587    * G') + (0.114    * B')
	//  Cb = 128 - (0.168736 * R') - (0.331264 * G') + (0.5      * B')
	//  Cr = 128 + (0.5      * R') - (0.418688 * G') - (0.081312 * B')
	//
	//  128.5 << 16 -> 257<<15
	y  = truncb((ir * 19595 + ig * 38470 + ib * 7471) >> 16);
	cb = truncb((-11056*ir - 21712*ig + 32768*ib + (257<<15)) >> 16);
	cr = truncb((32768*ir - 27440*ig - 5328*ib + (257<<15)) >> 16);
}

inline void FromYCbCr(uint8_t& r, uint8_t& g, uint8_t& b, uint8_t y, uint8_t cb, uint8_t cr) {
	//	R = Y' + 1.402   * (Cr-128)
	//	G = Y' - 0.34414 * (Cb-128) - 0.71414 * (Cr-128)
	//	B = Y' + 1.772   * (Cb-128)
	int iy = (int(y)<<16) + (1<<15);
	int icr = int(cr) - 128;
	int icb = int(cb) - 128;
	r = truncb((iy + 91881*icr) >> 16);
	g = truncb((iy - 22554*icb - 46802*icr) >> 16);
	b = truncb((iy + 116130*icb) >> 16);
}

void ColorizerIconEngine::Colorize(uint8_t &r, uint8_t &g, uint8_t &b) {
	if (!grayMode) {
		if (!darkMode) {
			return;
		}

		switch (colorMode) {

		case 0: { // HSL
			float h, s, l;
			ToHSL(h, s, l, r, g, b);
			l = 1.f - l;
			FromHSL(r, g, b, h, s, l);
			break;
		}

		case 1: { // Y'CbCr
			uint8_t y, cb, cr;
			ToYCbCr(y, cb, cr, r, g, b);
			y = 255-y;
			FromYCbCr(r, g, b, y, cb, cr);
			break;
		}

		case 2: { // CIE-L*ab
			colorspace::sRGB s{r, g, b};
			auto c = colorspace::Lab::from(s);

			c.L = 100.0 - c.L;

			s = colorspace::sRGB::from(c);
			r = s.r;
			g = s.g;
			b = s.b;
		}

		}
		return;
	}

	uint8_t y = gray(r, g, b);

	if (!darkMode) {
		y = 128 + y/4;
	} else {
		y = 64 + (255-y)/4;
	}

	r = y;
	g = y;
	b = y;
}

void Window::OnPaint(HDC dc, int dx, int dy) {
	RECT rect{0, 0, dx, dy};

	COLORREF bkcolor = eng->darkMode ? eng->darkbk : eng->lightbk;
	if (brushPattern) {
		::SetBkColor(dc, bkcolor);
		HBRUSH hbr = ::CreateHatchBrush(HS_DIAGCROSS, RGB(128, 128, 128));
		::FillRect(dc, &rect, hbr);
		::DeleteObject(hbr);
	} else {
		HBRUSH hbr = ::CreateSolidBrush(bkcolor);
		::FillRect(dc, &rect, hbr);
		::DeleteObject(hbr);
	}

	static constexpr int pad = 8;
	int x = pad, y = pad;

	int paintSize = paintSizes[paintSizeIdx];

	eng->DebugSinglePath(debugPathIdx);

	int i = int(singleIconIdx) - 1;
	if (i >= 0) {
		auto it = pack.begin() + i;
		RECT r{x, y, x+paintSize, y+paintSize};
		eng->DrawIconDirect(dc, &r, *it);
		return;
	}

	for (auto const& icon : pack) {
		RECT r{x, y, x+paintSize, y+paintSize};
		eng->DrawIconDirect(dc, &r, icon);

		x += paintSize + pad;
		if (x + paintSize > dx) {
			x = pad;
			y += paintSize + pad;
			if (y > dy) {
				return;
			}
		}
	}
}

void Window::Invalidate() {
	InvalidateRect(hwnd, nullptr, TRUE);
}

