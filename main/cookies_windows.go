//go:build windows && !no_cookies_detected

package main

import (
	"fmt"
	"github.com/elvis972602/kemono-scraper/main/cookie"
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
	if runtime.GOOS != "windows" {
		log.Fatalf("Cookies detected, but not supported on %s", runtime.GOOS)
	}

	var cookies []*http.Cookie
	c := cookie.NewCookies()
	err := c.ReadCookies(cookieBrowser, "", 0)
	if err != nil {
		log.Fatalf("Error reading cookies: %s", err)
	}
	cs := c.GetCookies()
	for _, v := range cs {
		if v.Domain == fmt.Sprintf("%s.party", s) || v.Domain == fmt.Sprintf(".%s.party", s) {
			cookies = append(cookies, v)
		}
	}
	return cookies
}
