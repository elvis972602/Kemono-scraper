//go:build windows

package term

import (
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/windows"
	"io"
	"strings"
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

func isPipe(fd uintptr) bool {
	typ, err := windows.GetFileType(windows.Handle(fd))
	return err == nil && typ == windows.FILE_TYPE_PIPE
}

func getFileNameByHandle(fd uintptr) (string, error) {
	type FILE_NAME_INFO struct {
		FileNameLength int32
		FileName       [windows.MAX_LONG_PATH]uint16
	}

	var fi FILE_NAME_INFO
	err := windows.GetFileInformationByHandleEx(windows.Handle(fd), windows.FileNameInfo, (*byte)(unsafe.Pointer(&fi)), uint32(unsafe.Sizeof(fi)))
	if err != nil {
		return "", err
	}

	filename := syscall.UTF16ToString(fi.FileName[:])
	return filename, nil
}

func CanUpdateStatus(fd uintptr) bool {
	// easy case, the terminal is cmd or psh, without redirection
	if isWindowsTerminal(fd) {
		return true
	}

	// pipes require special handling
	if !isPipe(fd) {
		return false
	}

	fn, err := getFileNameByHandle(fd)
	if err != nil {
		return false
	}

	// inspired by https://github.com/RyanGlScott/mintty/blob/master/src/System/Console/MinTTY/Win32.hsc
	// terminal: \msys-dd50a72ab4668b33-pty0-to-master
	// pipe to cat: \msys-dd50a72ab4668b33-13244-pipe-0x16
	if (strings.HasPrefix(fn, "\\cygwin-") || strings.HasPrefix(fn, "\\msys-")) &&
		strings.Contains(fn, "-pty") && strings.HasSuffix(fn, "-master") {
		return true
	}

	return false
}

func SupportsEscapeCodes(fd uintptr) bool {
	if isWindowsTerminal(fd) {
		h := syscall.Handle(fd)
		var mode uint32
		err := syscall.GetConsoleMode(h, &mode)
		if err != nil {
			return false
		}
		return mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0
	}
	return true
}
