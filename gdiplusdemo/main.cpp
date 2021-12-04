#ifndef UNICODE
#define UNICODE
#endif

#include "window.h"
#include "GdiPlusIcon.h"

#include <fstream>

LRESULT CALLBACK WindowProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam);

class IconWindow : public Window {
public:
	IconWindow(HINSTANCE inst, vectoricon::Pack const& pk) :
		Window(inst, L"Icon demo"),
		iconpack_(pk) {
	}

protected:
	LRESULT WindowProc(UINT msg, WPARAM wParam, LPARAM lParam) override;

	void OnKeyDown(WPARAM wParam);
	void OnPaint(HDC dc, int dx, int dy);

private:
	vectoricon::Pack const& iconpack_;
	int paintSize = 16;

	GdiPlusIconEngine eng;
};

int WINAPI wWinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance,
		PWSTR pCmdLine, int nCmdShow) {

	// Load icons
	vectoricon::Pack pk;

	std::ifstream strm("C:/src/rts/head/CLIENT/SRC/RES/rts.iconpk",
			std::ifstream::in | std::ifstream::binary);
	if (!pk.load(strm)) {
		return 0;
	}

	// Initialize GDI+.
	Gdiplus::GdiplusStartupInput gdiplusStartupInput;
	ULONG_PTR           gdiplusToken;
	Gdiplus::GdiplusStartup(&gdiplusToken, &gdiplusStartupInput, nullptr);

	IconWindow w(hInstance, pk);

    ShowWindow(w.Hwnd(), nCmdShow);
	UpdateWindow(w.Hwnd());

    // Run the message loop.
	Window::MessageLoop();

	Gdiplus::GdiplusShutdown(gdiplusToken);

	return 0;
}

LRESULT IconWindow::WindowProc(UINT msg, WPARAM wParam, LPARAM lParam) {
	switch (msg) {

    case WM_DESTROY:
        PostQuitMessage(0);
        return 0;

    case WM_KEYDOWN:
		if (wParam == VK_ESCAPE) {
			PostQuitMessage(0);
			return 0;
		}
		OnKeyDown(wParam);
		return 0;

    case WM_SIZE:
		InvalidateRect(Hwnd(), nullptr, TRUE);
		return 0;

    case WM_PAINT: {
		PAINTSTRUCT ps;
		HDC hdc = BeginPaint(Hwnd(), &ps);

		RECT rc;
		GetClientRect(Hwnd(), &rc);
		OnPaint(hdc, rc.right, rc.bottom);

		EndPaint(Hwnd(), &ps);
        return 0;
	}

    }

	return 0;
}

void IconWindow::OnKeyDown(WPARAM w) {
}

void IconWindow::OnPaint(HDC dc, int dx, int dy) {
	static constexpr int pad = 8;
	int x = pad, y = pad;
	for (auto const& icon : iconpack_) {
		RECT r{x, y, x+paintSize, y+paintSize};
		eng.DrawIcon(dc, &r, icon);

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
