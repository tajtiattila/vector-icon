
#include "window.h"

#include <unordered_map>
#include <stdexcept>

namespace detail {

std::unordered_map<HWND, Window*> windowMap;

LRESULT CALLBACK defWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
	auto it = windowMap.find(hwnd);
	if (it == windowMap.end()) {
		return DefWindowProc(hwnd, msg, wParam, lParam);
	}

	Window* w = it->second;
	LRESULT r = w->WindowProc(msg, wParam, lParam);

	if (msg == WM_DESTROY) {
		windowMap.erase(hwnd);
		delete w;
	}

	return r;
}

} // end namespace detail

Window::Window(HINSTANCE inst, LPCTSTR text) {
	static bool reg = false;

    // Register the window class.
	const wchar_t CLASS_NAME[] = L"Window";
	if (!reg) {
		reg = true;

		WNDCLASS wc = {};

		wc.lpfnWndProc    = detail::defWndProc;
		wc.hInstance      = inst;
		wc.lpszClassName  = CLASS_NAME;
		wc.hbrBackground  = (HBRUSH)GetStockObject(WHITE_BRUSH);
		wc.hCursor        = LoadCursor(nullptr, IDC_ARROW);

		RegisterClass(&wc);
	}

    HWND hwnd = CreateWindowEx(
        0,          // Optional window styles.
        CLASS_NAME, // Window class
        text,       // Window text
        WS_OVERLAPPEDWINDOW, // Window style

        // Size and position
        CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT,

        nullptr, // Parent window
        nullptr, // Menu
        inst,    // Instance handle
        nullptr  // Additional application data
        );
	if (!hwnd) {
		throw std::runtime_error("error creating window");
	}

	detail::windowMap.insert({hwnd, this});
	hwnd_ = hwnd;
}

Window::~Window() {
}

LRESULT Window::WindowProc(UINT msg, WPARAM wParam, LPARAM lParam) {
	return DefWindowProc(Hwnd(), msg, wParam, lParam);
}

void Window::MessageLoop() {
    MSG msg = {};
    while (GetMessage(&msg, NULL, 0, 0)) {
        TranslateMessage(&msg);
        DispatchMessage(&msg);
    }
}
