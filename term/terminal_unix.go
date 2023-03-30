//go:build !windows

package term

import (
	"io"
	"os"

	"golang.org/x/term"
)

func clearCurrentLine(wr io.Writer, fd uintptr) func(io.Writer, uintptr) {
	return posixClearCurrentLine
}

func moveCursorUp(wr io.Writer, fd uintptr) func(io.Writer, uintptr, int) {
	return posixMoveCursorUp
}

func CanUpdateStatus(fd uintptr) bool {
	if !term.IsTerminal(int(fd)) {
		return false
	}
	term := os.Getenv("TERM")
	if term == "" {
		return false
	}
	// TODO actually read termcap db and detect if terminal supports what we need
	return term != "dumb"
}

func SupportsEscapeCodes(fd uintptr) bool {
	return true
}
