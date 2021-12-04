#ifndef WINDOW_H_INCLUDED_
#define WINDOW_H_INCLUDED_

#define WIN32_LEAN_AND_MEAN
#include <windows.h>

class Window {
public:
	Window(HINSTANCE inst, LPCTSTR text);

	virtual ~Window();

	virtual LRESULT WindowProc(UINT msg, WPARAM wParam, LPARAM lParam);

	HWND Hwnd() const { return hwnd_; }

	static void MessageLoop();

private:
	HWND hwnd_;
};

#endif // WINDOW_H_INCLUDED_
