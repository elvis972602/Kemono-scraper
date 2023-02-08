package linux

import (
	"errors"
	secret_service "github.com/zalando/go-keyring/secret_service"
	"log"
	"os"
	"os/exec"
	"strings"
)

type LinuxDesktopEnvironment int

const (
	Other LinuxDesktopEnvironment = iota
	Unity
	Gnome
	Cinnamon
	Kde
	Pantheon
	Xfce
)

func GetLinuxDesktopEnvironment() LinuxDesktopEnvironment {
	xdgCurrentDesktop := os.Getenv("XDG_CURRENT_DESKTOP")
	desktopSession := os.Getenv("DESKTOP_SESSION")

	if xdgCurrentDesktop != "" {
		splits := strings.Split(xdgCurrentDesktop, ":")
		xdgCurrentDesktop = splits[0]
		xdgCurrentDesktop = strings.TrimSpace(xdgCurrentDesktop)

		if xdgCurrentDesktop == "Unity" {
			if desktopSession != "" && strings.Contains(desktopSession, "gnome-fallback") {
				return Gnome
			}
			return Unity
		} else if xdgCurrentDesktop == "GNOME" {
			return Gnome
		} else if xdgCurrentDesktop == "X-Cinnamon" {
			return Cinnamon
		} else if xdgCurrentDesktop == "KDE" {
			return Kde
		} else if xdgCurrentDesktop == "Pantheon" {
			return Pantheon
		} else if xdgCurrentDesktop == "XFCE" {
			return Xfce
		}
	} else if desktopSession != "" {
		if desktopSession == "mate" || desktopSession == "gnome" {
			return Gnome
		} else if strings.Contains(desktopSession, "kde") {
			return Kde
		} else if strings.Contains(desktopSession, "xfce") {
			return Xfce
		}
	} else {
		if os.Getenv("GNOME_DESKTOP_SESSION_ID") != "" {
			return Gnome
		} else if os.Getenv("KDE_FULL_SESSION") != "" {
			return Kde
		}
	}

	return Other
}

type LinuxKeyring int

const (
	// LinuxChromeCookieDecryptor is a decryptor for Linux Chrome cookies
	KWALLET LinuxKeyring = iota
	GNOMEKEYRING
	BASICTEXT
)

var LinuxKeyringNames = map[LinuxKeyring]bool{
	KWALLET:      true,
	GNOMEKEYRING: true,
	BASICTEXT:    true,
}

func ChooseLinuxKeyring() LinuxKeyring {
	var keyring LinuxKeyring
	env := GetLinuxDesktopEnvironment()
	if env == Kde {
		keyring = KWALLET
	} else if env == Other {
		keyring = BASICTEXT
	} else {
		keyring = GNOMEKEYRING
	}
	return keyring
}

func getKwalletNetworkWallet() string {
	// https://chromium.googlesource.com/chromium/src/+/refs/heads/main/components/os_crypt/kwallet_dbus.cc
	// KWalletDBus::NetworkWallet
	// https://api.kde.org/frameworks/kwallet/html/classKWallet_1_1Wallet.html
	// Wallet::NetworkWallet
	defaultWallet := "kdewallet"
	out, err := exec.Command("dbus-send", "--session", "--print-reply=literal",
		"--dest=org.kde.kwalletd5",
		"/modules/kwalletd5",
		"org.kde.KWallet.networkWallet").Output()
	if err != nil {
		log.Println("failed to read NetworkWallet")
		return defaultWallet
	}
	log.Println("NetworkWallet =", string(out))
	return strings.TrimSpace(string(out))
}

func GetKWalletPassword(browserKeyringName string) ([]byte, error) {
	log.Printf("GetKWalletPassword(%s)", browserKeyringName)

	networkWallet := getKwalletNetworkWallet()
	out, err := exec.Command("kwallet-query", "--read-password", browserKeyringName+" Safe Storage", "--folder", browserKeyringName+" Keys", networkWallet).Output()
	if err != nil {
		return nil, errors.New("failed to read password")
	}
	log.Println("password =", string(out))
	return []byte(strings.TrimSpace(string(out))), nil
}

func GetGnomeKeyringPassword(browserKeyringName string) ([]byte, error) {
	svc, err := secret_service.NewSecretService()
	if err != nil {
		return nil, errors.New("failed to create secret service")
	}
	collection := svc.GetLoginCollection()
	if err = svc.Unlock(collection.Path()); err != nil {
		return nil, errors.New("failed to unlock login collection")
	}

	items, err := svc.SearchItems(collection, map[string]string{
		"application": browserKeyringName,
	})
	if err != nil {
		return nil, errors.New("failed to search items")
	}
	if len(items) == 0 {
		return nil, errors.New("no items found")
	}
	item := items[0]

	session, err := svc.OpenSession()
	if err != nil {
		return nil, err
	}
	defer svc.Close(session)

	secret, err := svc.GetSecret(item, session.Path())
	if err != nil {
		return nil, err
	}
	return secret.Value, nil
}
