//go:build !windows

package main

import (
	"log"
	"net/http"
	"os"
	"runtime"
)

func getCookies(s string) []*http.Cookie {
	if cookieFile != "" {
		f, err := os.Stat(cookieFile)
		if err != nil && f != nil {
			log.Printf("load cookie from %s", cookieFile)
			return parasCookeiFile(cookieFile)
		}
	}
	log.Fatalf("Cookies detected, but not supported on %s", runtime.GOOS)
	return nil
}
