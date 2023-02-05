//go:build windows

package term

import (
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/windows"
	"io"
	"syscall"
	"unsafe"
)

func clearCurrentLine(wr io.Writer, fd uintptr) func(io.Writer, uintptr) {
	// easy case, the terminal is cmd or psh, without redirection
	if isWindowsTerminal(fd) {
		return windowsClearCurrentLine
	}

	// assume we're running in mintty/cygwin
	return posixClearCurrentLine
}

// moveCursorUp moves the cursor to the line n lines above the current one.
func moveCursorUp(wr io.Writer, fd uintptr) func(io.Writer, uintptr, int) {
	// easy case, the terminal is cmd or psh, without redirection
	if isWindowsTerminal(fd) {
		return windowsMoveCursorUp
	}

	// assume we're running in mintty/cygwin
	return posixMoveCursorUp
}

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

var (
	procFillConsoleOutputCharacter = kernel32.NewProc("FillConsoleOutputCharacterW")
	procFillConsoleOutputAttribute = kernel32.NewProc("FillConsoleOutputAttribute")
)

func windowsClearCurrentLine(wr io.Writer, fd uintptr) {
	var info windows.ConsoleScreenBufferInfo
	err := windows.GetConsoleScreenBufferInfo(windows.Handle(fd), &info)
	if err != nil {
		panic(err)
	}

	// clear the line
	cursor := windows.Coord{
		X: info.Window.Left,
		Y: info.CursorPosition.Y,
	}
	var count, w uint32
	count = uint32(info.Size.X)
	procFillConsoleOutputAttribute.Call(fd, uintptr(info.Attributes), uintptr(count), *(*uintptr)(unsafe.Pointer(&cursor)), uintptr(unsafe.Pointer(&w)))
	procFillConsoleOutputCharacter.Call(fd, uintptr(' '), uintptr(count), *(*uintptr)(unsafe.Pointer(&cursor)), uintptr(unsafe.Pointer(&w)))
}

func windowsMoveCursorUp(wr io.Writer, fd uintptr, n int) {
	var info windows.ConsoleScreenBufferInfo
	windows.GetConsoleScreenBufferInfo(windows.Handle(fd), &info)

	// move cursor up by n lines and to the first column
	windows.SetConsoleCursorPosition(windows.Handle(fd), windows.Coord{
		X: 0,
		Y: info.CursorPosition.Y - int16(n),
	})

}

func isWindowsTerminal(fd uintptr) bool {
	return terminal.IsTerminal(int(fd))
}
