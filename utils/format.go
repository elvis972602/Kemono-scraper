package utils

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	B = 1 << (10 * iota)
	KB
	MB
	GB
	TB
)

const (
	Nanosecond  = 1
	Microsecond = 1000 * Nanosecond
	Millisecond = 1000 * Microsecond
	Second      = 1000 * Millisecond
	Minute      = 60 * Second
	Hour        = 60 * Minute
)

func FormatSize(size int64) string {

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2fTB", float64(size)/TB)
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

func ParseSize(size string) int64 {
	size = strings.ToUpper(size)
	var (
		fvalue float64
		unit   string
		value  string
	)
	// 1MB or 1 MB
	parts := strings.Fields(size)
	if len(parts) != 2 {
		for _, u := range []string{"TB", "GB", "MB", "KB", "B"} {
			if strings.HasSuffix(size, u) {
				value = strings.TrimSuffix(size, u)
				unit = u
				break
			}
		}
		fvalue, _ = strconv.ParseFloat(value, 64)
	} else {
		fvalue, _ = strconv.ParseFloat(parts[0], 64)
		unit = parts[1]
	}
	switch unit {
	case "TB":
		return int64(fvalue * TB)
	case "GB":
		return int64(fvalue * GB)
	case "MB":
		return int64(fvalue * MB)
	case "KB":
		return int64(fvalue * KB)
	default:
		return int64(fvalue)
	}
}

func FormatDuration(duration int64) string {
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
