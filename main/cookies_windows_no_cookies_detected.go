//go:build windows && no_cookies_detection

package main

import (
	"log"
	"net/http"
	"os"
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
	return []*http.Cookie{}
}
