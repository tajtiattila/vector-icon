#ifndef UNICODE
#define UNICODE
#endif

#define WIN32_LEAN_AND_MEAN
#include <windows.h>

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
	bool direct_ = false;
};

std::vector<int> Window::paintSizes = {16, 20, 24, 28, 32, 48, 64, 128};

Window g_window;

LRESULT CALLBACK WindowProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

int WINAPI wWinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance,
		PWSTR pCmdLine, int nCmdShow) {

	{
		std::string fn;
		wchar_t* p = pCmdLine;
		while (*p != 0) {
			fn.push_back((char)(uint8_t)(*p++));
		}
		if (fn.empty()) {
			MessageBox(nullptr, L"No icon pack specified.", L"Error", MB_OK|MB_ICONSTOP);
			return 0;
		}
		std::ifstream strm(fn, std::ifstream::in | std::ifstream::binary);
		if (!strm.good()) {
			MessageBox(nullptr, L"Error opening icon pack.", L"Error", MB_OK|MB_ICONSTOP);
			return 0;
		}

		if (!g_window.pack.load(strm)) {
			MessageBox(nullptr, L"Error loading icon pack.", L"Error", MB_OK|MB_ICONSTOP);
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

	case 'X':
		direct_ = !direct_;
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

	case 'Q':
		eng->grayMode = !eng->grayMode;
		break;

	case 'E':
		eng->darkMode = !eng->darkMode;
		break;

	case 'R':
		eng->colorMode = (eng->colorMode + 1) % 3;
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

void ColorizerIconEngine::Colorize(uint8_t &r, uint8_t &g, uint8_t &b) {
	if (!grayMode) {
		return;
	}

	uint8_t y = gray(r, g, b);

	if (!darkMode) {
		y = 128 + y/4;
	} else {
		y = 64 + y/4;
	}

	r = y;
	g = y;
	b = y;
}

void Window::OnPaint(HDC dc, int dx, int dy) {
	RECT rect{0, 0, dx, dy};

	COLORREF bkcolor = eng->darkMode ? eng->darkbk : eng->lightbk;
	size_t palidx = eng->darkMode ? 1 : 0;
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
		eng->DrawIconEx(direct_, dc, &r, *it, palidx);
		return;
	}

	for (auto const& icon : pack) {
		RECT r{x, y, x+paintSize, y+paintSize};
		eng->DrawIconEx(direct_, dc, &r, icon, palidx);

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

