package cookie

import (
	"testing"
)

func Test_Chrome_Cookies(t *testing.T) {
	c := NewCookies()
	err := c.ReadCookies("chrome", "", 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	_ = c.GetCookies()
}

func Test_Firefox_Cookies(t *testing.T) {
	c := NewCookies()
	err := c.ReadCookies("firefox", "", 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	_ = c.GetCookies()
}

func Test_Opera_Cookies(t *testing.T) {
	c := NewCookies()
	err := c.ReadCookies("opera", "", 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	_ = c.GetCookies()
}

func Test_Edge_Cookies(t *testing.T) {
	c := NewCookies()
	err := c.ReadCookies("edge", "", 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	_ = c.GetCookies()
}
