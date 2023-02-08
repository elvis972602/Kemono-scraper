package utils

import "runtime"

var ChromiumBasedBrowsers = map[string]bool{
	"brave":    true,
	"chrome":   true,
	"chromium": true,
	"edge":     true,
	"opera":    true,
	"vivaldi":  true,
}

var SupportedBrowsers = map[string]bool{
	"brave":    true,
	"chrome":   true,
	"chromium": true,
	"edge":     true,
	"opera":    true,
	"vivaldi":  true,
	"firefox":  true,
	"safari":   true,
}

var WindowsBrowserDir = map[string]string{
	"brave":    "BraveSoftware\\Brave-Browser\\User Data",
	"chrome":   "Google\\Chrome\\User Data",
	"chromium": "Chromium\\User Data",
	"edge":     "Microsoft\\Edge\\User Data",
	"opera":    "Opera Software\\Opera Stable",
	"vivaldi":  "Vivaldi\\User Data",
}

var DarwinBrowserDir = map[string]string{
	"brave":    "BraveSoftware\\Brave-Browser",
	"chrome":   "Google\\Chrome",
	"chromium": "Chromium",
	"edge":     "Microsoft Edge",
	"opera":    "com.operasoftware.Opera",
	"vivaldi":  "Vivaldi",
}

var LinuxBrowserDir = map[string]string{
	"brave":    "BraveSoftware\\Brave-Browser",
	"chrome":   "google-chrome",
	"chromium": "chromium",
	"edge":     "microsoft-edge",
	"opera":    "opera",
	"vivaldi":  "vivaldi",
}

func KeyingName(browserName string) string {
	switch browserName {
	case "brave":
		return "Brave"
	case "chrome":
		return "Chrome"
	case "chromium":
		return "Chromium"
	case "edge":
		if runtime.GOOS == "darwin" {
			return "Chromium"
		} else {
			return "Microsoft Edge"
		}
	case "opera":
		if runtime.GOOS == "darwin" {
			return "Chromium"
		} else {
			return "Opera"
		}
	case "vivaldi":
		if runtime.GOOS == "darwin" {
			return "Chrome"
		} else {
			return "Chromium"
		}
	default:
		panic("KeyingName called with unsupported browser " + browserName)
	}
}
