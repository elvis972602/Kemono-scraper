//go:build windows && !no_cookies_detection

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/elvis972602/kemono-scraper/main/cookie"
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
		if v.Domain == fmt.Sprintf("%s.su", s) || v.Domain == fmt.Sprintf(".%s.su", s) {
			cookies = append(cookies, v)
		}
	}
	return cookies
}
