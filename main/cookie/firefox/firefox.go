package firefox

import (
	"os"
	"path/filepath"
	"runtime"
)

func BrowserDir() string {
	if runtime.GOOS == "windows" {
		appDataRoaming := os.Getenv("APPDATA")
		return filepath.FromSlash(filepath.Join(appDataRoaming, "Mozilla", "Firefox", "Profiles"))
	} else if runtime.GOOS == "darwin" {
		panic("TODO: implement darwin")
	}
	return filepath.FromSlash(os.ExpandEnv("~/.mozilla/firefox"))
}
