package cookie

import (
	"database/sql"
	"fmt"
	"github.com/elvis972602/kemono-scraper/main/cookie/chromium"
	"github.com/elvis972602/kemono-scraper/main/cookie/firefox"
	"github.com/elvis972602/kemono-scraper/main/cookie/utils"
	"github.com/elvis972602/kemono-scraper/main/cookie/utils/linux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Cookies struct {
	cookies []*http.Cookie
}

func NewCookies() *Cookies {
	return &Cookies{}
}

func (c *Cookies) GetCookies() []*http.Cookie {
	return c.cookies
}

func (c *Cookies) ReadCookies(browserName, profile string, keyring linux.LinuxKeyring) error {
	browserName, profile, keyring = parseBrowserSpecification(browserName, profile, keyring)
	return c.extractCookiesFromBrowser(browserName, profile, keyring)
}

func (c *Cookies) extractCookiesFromBrowser(browserName, profile string, keying linux.LinuxKeyring) error {
	if browserName == "firefox" {
		return c.extractFireFoxCookies(profile)
	} else if browserName == "safari" {
		// TODO: extract cookies from safari
		panic("TODO: implement safari")
	} else if utils.ChromiumBasedBrowsers[browserName] {
		return c.extractChromiumBasedCookies(browserName, profile, keying)
	} else {
		return fmt.Errorf("browser %s not supported", browserName)
	}
	return nil
}

func (c *Cookies) extractChromiumBasedCookies(browserName, profile string, keyring linux.LinuxKeyring) error {
	var searchRoot string
	browserDir, keyringName, supportsProfiles := chromium.GetChromiumBasedBrowserSettings(browserName)

	if profile == "" {
		searchRoot = browserDir
	} else if filepath.IsAbs(profile) {
		searchRoot = browserDir
		if supportsProfiles {
			browserDir = filepath.Dir(profile)
		} else {
			browserDir = profile
		}
	} else {
		if supportsProfiles {
			searchRoot = filepath.Join(browserDir, profile)
		} else {
			// no profiles, so profile is the path to the browser
			searchRoot = browserDir
		}
	}

	cookieDatabasePath := utils.FindMostRecentlyUsedFile(searchRoot, "Cookies")
	if cookieDatabasePath == "" {
		return fmt.Errorf("could not find cookies database for %s", browserName)
	}

	log.Println("found cookie database")

	decryptor, err := chromium.GetCookieDecryptor(searchRoot, keyringName, keyring)
	if err != nil {
		return fmt.Errorf("could not get cookie decryptor: %w", err)
	}

	tmpdir, err := os.MkdirTemp(".", "cookies")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	db, err := openDataBaseCopy(cookieDatabasePath, tmpdir)
	if err != nil {
		return fmt.Errorf("could not open database copy: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT host_key, name, value, encrypted_value, path, expires_utc, is_secure FROM cookies")
	if err != nil {
		return fmt.Errorf("could not query cookies: %w", err)
	}
	cookies := make([]*http.Cookie, 0)
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
		var (
			host, name, value, encryptedValue, p string
			expires, isSecure                    int
		)
		if err = rows.Scan(&host, &name, &value, &encryptedValue, &p, &expires, &isSecure); err != nil {
			return fmt.Errorf("could not scan row: %w", err)
		}
		_, cookie, err := processChromeCookie(decryptor, host, name, value, encryptedValue, p, expires, isSecure)
		if err != nil {
			err = fmt.Errorf("could not process cookie: %w", err)
			return err
		}
		cookies = append(cookies, cookie)
	}

	c.cookies = cookies

	return nil
}

func (c *Cookies) extractFireFoxCookies(profile string) error {
	var searchRoot string
	if profile == "" {
		searchRoot = firefox.BrowserDir()
	} else if filepath.IsAbs(profile) {
		searchRoot = profile
	} else {
		searchRoot = filepath.Join(firefox.BrowserDir(), profile)
	}
	cookieDatabasePath := utils.FindMostRecentlyUsedFile(searchRoot, "cookies.sqlite")
	if cookieDatabasePath == "" {
		return fmt.Errorf("could not find cookies database for firefox")
	}

	log.Println("found cookie database")

	tmpdir, err := os.MkdirTemp(".", "cookies")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	db, err := openDataBaseCopy(cookieDatabasePath, tmpdir)
	if err != nil {
		return fmt.Errorf("could not open database copy: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT host, name, value, path, expiry, isSecure FROM moz_cookies")
	if err != nil {
		return fmt.Errorf("could not query cookies: %w", err)
	}
	cookies := make([]*http.Cookie, 0)
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
		var (
			host, name, value, p string
			expires, isSecure    int
		)
		if err = rows.Scan(&host, &name, &value, &p, &expires, &isSecure); err != nil {
			return fmt.Errorf("could not scan row: %w", err)
		}
		cookie := &http.Cookie{
			Name:     name,
			Value:    value,
			Path:     p,
			Domain:   host,
			Secure:   isSecure == 1,
			HttpOnly: true,
			Expires:  time.Unix(int64(expires), 0),
		}
		cookies = append(cookies, cookie)
	}
	c.cookies = cookies
	return nil
}

func parseBrowserSpecification(browserName, profile string, keyring linux.LinuxKeyring) (string, string, linux.LinuxKeyring) {
	if !utils.SupportedBrowsers[browserName] {
		log.Fatal("browser " + browserName + " not supported")
	}
	if !linux.LinuxKeyringNames[keyring] {
		log.Fatal(fmt.Sprintf("keyring %d not supported", keyring))
	}
	if profile != "" {
		sbsPath, err := filepath.Abs(profile)
		if err != nil {
			profile = sbsPath
		}
	}
	return browserName, profile, keyring
}

type CookieDecryptor interface {
	Decrypt(encrypted []byte) ([]byte, error)
}

func openDataBaseCopy(dbPath, tmpdir string) (*sql.DB, error) {
	cpPath := path.Join(tmpdir, "temporary.sqlite")
	data, err := ioutil.ReadFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not read database file: %w", err)
	}

	err = ioutil.WriteFile(cpPath, data, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not write database file: %w", err)
	}
	return sql.Open("sqlite3", cpPath)
}

func processChromeCookie(decryptor CookieDecryptor, host_key, name, value, encryptedValue, path string, expires_utc, is_secure int) (isEncrypted bool, cookie *http.Cookie, err error) {
	isEncrypted = value == "" && encryptedValue != ""
	if isEncrypted {
		decryptedValue, err := decryptor.Decrypt([]byte(encryptedValue))
		if err != nil {
			return false, nil, fmt.Errorf("could not Decrypt cookie: %w", err)
		}
		value = string(decryptedValue)
	}
	cookie = &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   host_key,
		Secure:   is_secure == 1,
		HttpOnly: true,
		MaxAge:   expires_utc - int(time.Now().Unix()),
	}
	return
}
