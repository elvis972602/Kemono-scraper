//go:build linux

package main

import (
	"log"
	"net/http"
	"os"
	"runtime"
)

func getCookies(s string) []*http.Cookie {
	if cookieFile != "" {
		_, err := os.Stat(cookieFile)
		if err == nil {
			log.Printf("load cookie from %s", cookieFile)
			return parasCookieFile(cookieFile)
		} else {
			log.Printf("cookie file %s not found", cookieFile)
		}
	}
	log.Fatalf("Cookies detected, but not supported on %s", runtime.GOOS)
	return nil
}
