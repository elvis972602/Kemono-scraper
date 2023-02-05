package term

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const (
	posixControlMoveCursorHome = "\r"
	posixControlMoveCursorUp   = "\x1b[1A"
	posixControlClearLine      = "\x1b[2K"
)

func posixClearCurrentLine(wr io.Writer, fd uintptr) {
	// clear current line
	_, err := wr.Write([]byte(posixControlMoveCursorHome + posixControlClearLine))
	if err != nil {
		fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		return
	}
}

func posixMoveCursorUp(wr io.Writer, fd uintptr, n int) {
	data := []byte(posixControlMoveCursorHome)
	data = append(data, bytes.Repeat([]byte(posixControlMoveCursorUp), n)...)
	_, err := wr.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		return
	}
}
