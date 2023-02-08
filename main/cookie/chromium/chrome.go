package chromium

import (
	"github.com/elvis972602/kemono-scraper/main/cookie/utils"
	"github.com/elvis972602/kemono-scraper/main/cookie/utils/linux"
	"os"
	"path/filepath"
	"runtime"
)

type ChroniumBasedBrowser struct{}

type CookieDecryptor interface {
	Decrypt(encrypted []byte) ([]byte, error)
}

func GetCookieDecryptor(browserRoot, keyringName string, keyring linux.LinuxKeyring) (CookieDecryptor, error) {
	return NewChromeCookieDecryptor(browserRoot, keyringName, keyring)
}

func GetChromiumBasedBrowserSettings(browserName string) (browserDir string, keyringName string, supportsProfiles bool) {
	appDataLocal := os.Getenv("LOCALAPPDATA")
	appDataRoaming := os.Getenv("APPDATA")
	switch runtime.GOOS {
	case "windows":
		if browserName == "opera" {
			browserDir = filepath.Join(appDataRoaming, utils.WindowsBrowserDir[browserName])
		} else {
			browserDir = filepath.Join(appDataLocal, utils.WindowsBrowserDir[browserName])
		}
	case "darwin":
		panic("TODO: implement darwin")
	default:
		browserDir = filepath.Join(utils.ConfigHome(), utils.LinuxBrowserDir[browserName])
	}

	keyringName = utils.KeyingName(browserName)
	supportsProfiles = browserName != "opera"
	return
}
