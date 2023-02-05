package utils

import (
	"fmt"
	"golang.org/x/term"
	"golang.org/x/text/width"
	"syscall"
	"unicode"
)

func FormatSize(size int64) string {
	const KB = 1024
	const MB = KB * 1024
	const GB = MB * 1024

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func FormatDuration(duration int64) string {
	const (
		Nanosecond  = 1
		Microsecond = 1000 * Nanosecond
		Millisecond = 1000 * Microsecond
		Second      = 1000 * Millisecond
		Minute      = 60 * Second
		Hour        = 60 * Minute
	)

	switch {
	case duration >= Hour:
		return fmt.Sprintf("%.2dh%.2fm", duration/Hour, float64(duration%Hour)/Minute)
	case duration >= Minute:
		return fmt.Sprintf("%.2dm%.2fs", duration/Minute, float64(duration%Minute)/Second)
	case duration >= Second:
		return fmt.Sprintf("%.2fs", float64(duration)/Second)
	default:
		return fmt.Sprintf("%.2fms", float64(duration)/Millisecond)
	}
}

func wideRune(r rune) bool {
	kind := width.LookupRune(r).Kind()
	return kind != width.Neutral && kind != width.EastAsianNarrow
}

func truncate(s string, w int) int {
	if len(s) < w {
		// Since the display width of a character is at most 2
		// and all of ASCII (single byte per rune) has width 1,
		// no character takes more bytes to encode than its width.
		return w
	}

	var (
		i int
		r rune
	)

	for i, r = range s {
		w--
		if r > unicode.MaxASCII && wideRune(r) {
			w--
		}

		if w < 0 {
			break
		}
	}

	return i
}

func ShortenString(s1, s2, s3 string) string {
	return shortenString(s1+s2+s3, len(s1), len(s1)+len(s2))
}

func shortenString(str string, start, end int) string {
	width, _, err := term.GetSize(int(syscall.Stdout))
	if err != nil || width <= 0 {
		// use 80 columns by default
		width = 80
	}

	var length int

	if width > 2 {
		length = truncate(str, width-2)
	} else {
		length = width
	}

	if str == "" {
		return ""
	}
	if len(str) <= length {
		return str
	}
	if length < 3 {
		return "..."
	}

	remain := len(str) - length + 3
	sub := end - start + 1 - 3
	characters := []rune(str)
	mid := start + (end-start)/2
	end = mid
	start = mid

	for i := 0; i < remain; i++ {
		if i > sub {
			if end < len(characters) {
				end++
			} else {
				start--
			}
		} else {
			if i%2 == 0 {
				if start > 0 {
					start--
				} else {
					end++
				}
			} else {
				if end < len(characters) {
					end++
				} else {
					start--
				}
			}
		}

	}
	shortStr := string(characters[:start]) + "..." + string(characters[end:])
	return shortStr
}
