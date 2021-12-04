#ifndef UNICODE
#define UNICODE
#endif

#define WIN32_LEAN_AND_MEAN
#include <windows.h>

#include "GdiPlusIcon.h"

#include <fstream>

class Window {
public:
	void Invalidate();

	void OnKeyDown(WPARAM);
	void OnPaint(HDC dc, int dx, int dy);

	HWND hwnd;
	GdiPlusIconEngine* eng;
	vectoricon::Pack pack;

	size_t paintSizeIdx = 0;
	static std::vector<int> paintSizes;
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
	g_window.eng = new GdiPlusIconEngine;

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

	case 'J':
		paintSizeIdx++;
		if (paintSizeIdx >= paintSizes.size()) {
			paintSizeIdx = 0;
		}
		Invalidate();
		break;

	case 'K':
		paintSizeIdx--;
		if (paintSizeIdx >= paintSizes.size()) {
			paintSizeIdx = paintSizes.size() - 1;
		}
		Invalidate();
		break;

	}
}

void Window::OnPaint(HDC dc, int dx, int dy) {
	static constexpr int pad = 8;
	int x = pad, y = pad;

	int paintSize = paintSizes[paintSizeIdx];

#if 0
	RECT r{x, y, x+paintSize, y+paintSize};
	eng->DrawIconDirect(dc, &r, *pack.begin());
	return;
#endif

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
