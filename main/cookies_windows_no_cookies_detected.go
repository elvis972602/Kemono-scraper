//go:build windows && no_cookies_detected

package main

import (
	"log"
	"net/http"
	"os"
)

func getCookies(s string) []*http.Cookie {
	if cookieFile != "" {
		f, err := os.Stat(cookieFile)
		if err != nil && f != nil {
			log.Printf("load cookie from %s", cookieFile)
			return parasCookieFile(cookieFile)
		}
	}
	return []*http.Cookie{}
}
