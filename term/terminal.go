package term

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"golang.org/x/term"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const asciiEscapeCode = "\x1b[0m"

// print directly to the terminal
type message struct {
	line string
	err  bool
}

// update lines in the terminal
type status struct {
	lines []string
}

type fder interface {
	Fd() uintptr
}

// no implementation for background mode
type Terminal struct {
	wr        *bufio.Writer
	fd        uintptr
	errWriter io.Writer
	buf       *bytes.Buffer
	msg       chan message
	status    chan status
	// if it has the fd, can update the status lines
	updateStatus bool

	// will be closed when the goroutine which runs Run() terminates, so it'll
	// yield a default value immediately
	closed chan struct{}

	clearCurrentLine func(io.Writer, uintptr)
	moveCursorUp     func(io.Writer, uintptr, int)
	asciiResetCode   string
}

func NewTerminal(w io.Writer, errWriter io.Writer, disableStatus bool) *Terminal {
	t := &Terminal{
		wr:        bufio.NewWriter(w),
		errWriter: errWriter,
		buf:       bytes.NewBuffer(nil),
		msg:       make(chan message),
		status:    make(chan status),
		closed:    make(chan struct{}),
	}
	if disableStatus {
		return t
	}

	// if it has the fd, can update the status lines
	if f, ok := w.(fder); ok && CanUpdateStatus(f.Fd()) {
		t.updateStatus = true
		t.fd = f.Fd()
		t.clearCurrentLine = clearCurrentLine(w, t.fd)
		t.moveCursorUp = moveCursorUp(w, t.fd)
	}

	// check if the terminal supports ascii escape codes
	if t.updateStatus && SupportsEscapeCodes(t.fd) {
		t.asciiResetCode = asciiEscapeCode
	} else {
		t.asciiResetCode = ""
	}

	return t
}

func (t *Terminal) Run(ctx context.Context) {
	defer close(t.closed)
	if t.updateStatus {
		t.run(ctx)
		return
	}

	t.runWithoutStatus(ctx)
}

// run listens on the channels and updates the terminal screen.
func (t *Terminal) run(ctx context.Context) {
	var status []string
	var lastLineCount int
	for {
		select {
		case <-ctx.Done():

			return

		case msg := <-t.msg:
			for i := 0; i < lastLineCount-1; i++ {
				t.clearCurrentLine(t.wr, t.fd)
				t.moveCursorUp(t.wr, t.fd, 1)
			}
			t.clearCurrentLine(t.wr, t.fd)
			io.Writer(t.wr).Write([]byte("\r"))

			var dst io.Writer
			if msg.err {
				dst = t.errWriter

				// assume t.wr and t.errWriter are different, so we need to
				// flush clearing the current line
				err := t.wr.Flush()
				if err != nil {
					fmt.Fprintf(os.Stderr, "flush failed: %v\n", err)
				}
			} else {
				dst = t.wr
			}

			if _, err := io.WriteString(dst, msg.line); err != nil {
				fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
				continue
			}

			t.writeStatus(status)

			if err := t.wr.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "flush failed: %v\n", err)
			}

		case stat := <-t.status:
			for i := 0; i < lastLineCount-1; i++ {
				t.clearCurrentLine(t.wr, t.fd)
				t.moveCursorUp(t.wr, t.fd, 1)
			}
			t.clearCurrentLine(t.wr, t.fd)
			io.Writer(t.wr).Write([]byte("\r"))
			lastLineCount = len(stat.lines)
			status = status[:0]
			status = append(status, stat.lines...)
			t.writeStatus(status)
		}
	}
}

func (t *Terminal) writeStatus(status []string) {
	for _, line := range status {
		//t.clearCurrentLine(t.wr, t.fd)

		_, err := t.wr.WriteString(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		}

		// flush is needed so that the current line is updated
		err = t.wr.Flush()
		if err != nil {
			fmt.Fprintf(os.Stderr, "flush failed: %v\n", err)
		}
	}

	err := t.wr.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "flush failed: %v\n", err)
	}
}

func (t *Terminal) runWithoutStatus(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-t.msg:
			var flush func() error

			var dst io.Writer
			if msg.err {
				dst = t.errWriter
			} else {
				dst = t.wr
				flush = t.wr.Flush
			}

			if _, err := io.WriteString(dst, msg.line); err != nil {
				fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
			}

			if flush == nil {
				continue
			}

			if err := flush(); err != nil {
				fmt.Fprintf(os.Stderr, "flush failed: %v\n", err)
			}

		case stat := <-t.status:
			for _, line := range stat.lines {
				// Ensure that each message ends with exactly one newline.
				fmt.Fprintln(t.wr, strings.TrimRight(line, "\n"))
			}
			if err := t.wr.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "flush failed: %v\n", err)
			}
		}
	}
}

// clear the previously written status lines
func (t *Terminal) undoStatus(lines int) {
	for i := 0; i < lines; i++ {
		t.clearCurrentLine(t.wr, t.fd)

		_, err := t.wr.WriteRune('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		}

		// flush is needed so that the current line is updated
		err = t.wr.Flush()
		if err != nil {
			fmt.Fprintf(os.Stderr, "flush failed: %v\n", err)
		}
	}

	t.moveCursorUp(t.wr, t.fd, lines)

	err := t.wr.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "flush failed: %v\n", err)
	}
}

func (t *Terminal) print(line string, isErr bool) {
	// make sure the line ends with a line break
	if line[len(line)-1] != '\n' {
		line += "\n"
	}

	select {
	case t.msg <- message{line: line, err: isErr}:
	case <-t.closed:
	}
}

// Print writes a line to the terminal.
func (t *Terminal) Print(line string) {
	t.print(line, false)
}

// Printf uses fmt.Sprintf to write a line to the terminal.
func (t *Terminal) Printf(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)
	t.Print(s)
}

// Error writes an error to the terminal.
func (t *Terminal) Error(line string) {
	t.print(line, true)
}

// Errorf uses fmt.Sprintf to write an error line to the terminal.
func (t *Terminal) Errorf(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)
	t.Error(s)
}

var asciiColorPat = regexp.MustCompile(`(\x1b\[[0-9;]*m)?([\s\S]*?)(\x1b\[[0-9;]*m|$)`)

// Truncate truncates a string to a given width, taking into account ANSI color
func Truncate(s string, w int) string {
	var (
		builder strings.Builder
	)
	for _, m := range asciiColorPat.FindAllStringSubmatchIndex(s, -1) {
		for i := 0; i < len(m); i++ {
			if m[i] == -1 {
				m[i] = 0
			}
		}
		// write ascii color
		builder.WriteString(s[m[2]:m[3]])
		// get text length
		textLen := utf8.RuneCountInString(s[m[4]:m[5]])
		if textLen > w {
			cutRune := truncateString(s[m[4]:m[5]], w)
			builder.Write([]byte(string(cutRune)))
			break
		} else {
			builder.WriteString(s[m[4]:m[5]])
			w -= textLen
		}
		if len(m) == 8 {
			builder.WriteString(s[m[6]:m[7]])
		}
	}
	return builder.String()
}

func truncateString(runes string, w int) []rune {
	var cutRunes []rune
	for i := 0; i < len(runes); {
		_, size := utf8.DecodeRuneInString(runes[i:])
		if w < size {
			break
		}
		cutRunes = append(cutRunes, []rune(runes[i:i+size])...)
		w -= size
		i += size
	}
	return cutRunes
}

// SetStatus updates the status lines.
func (t *Terminal) SetStatus(lines []string) {
	if len(lines) == 0 {
		return
	}

	// only truncate interactive status output
	var width int
	if t.updateStatus {
		var err error
		width, _, err = term.GetSize(int(t.fd))
		if err != nil || width <= 0 {
			// use 80 columns by default
			width = 80
		}
	}

	// make sure that all lines have a line break and are not too long
	for i, line := range lines {
		line = strings.TrimRight(line, "\n")
		if width > 0 {
			line = Truncate(line, width-2)
		}
		lines[i] = line + t.asciiResetCode + "\n"
	}

	// make sure the last line does not have a line break
	last := len(lines) - 1
	lines[last] = strings.TrimRight(lines[last], "\n")

	select {
	case t.status <- status{lines: lines}:
	case <-t.closed:
	}
}
