//go:build windows

package chromium

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elvis972602/kemono-scraper/main/cookie/utils"
	"github.com/elvis972602/kemono-scraper/main/cookie/utils/linux"
	"log"
	"os"
)

type ChromeCookieDecryptor struct {
	v10Key []byte
	v10    int
	v11    int
}

func NewChromeCookieDecryptor(browserRoot, keyringName string, keyring linux.LinuxKeyring) (*ChromeCookieDecryptor, error) {
	k, err := getWindowsV10Key(browserRoot)
	if err != nil {
		return nil, err
	}
	return &ChromeCookieDecryptor{
		v10Key: k,
		v10:    0,
		v11:    0,
	}, nil
}

func (d *ChromeCookieDecryptor) Decrypt(encrypted []byte) ([]byte, error) {
	version := encrypted[:3]
	ciphertext := encrypted[3:]
	if bytesEqual(version, []byte("v10")) {
		// vxx (3 bytes)
		// nonce (12 bytes)
		// ciphertext (variable)
		// tag (16 bytes)
		d.v10++
		if d.v10Key == nil {
			return nil, errors.New("v10 key is nil")
		}
		nonceLength := 96 / 8
		authenticationTagLength := 16

		rawCiphertext := ciphertext
		nonce := rawCiphertext[:nonceLength]
		ciphertextWithTag := rawCiphertext[nonceLength:]
		// authenticationTag 16 bytes
		_ = ciphertextWithTag[len(ciphertext)-authenticationTagLength:]

		return decryptAESGCM(ciphertextWithTag, d.v10Key, nonce)
	} else {
		d.v11++
		return utils.DecryptWindowsDpapi(encrypted)
	}
}

// byte array equal
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func decryptAESGCM(ciphertext, key, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("could not create AES cipher: %v", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not create AES GCM: %v", err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("could not Decrypt AES GCM: %v", err)
	}

	return plaintext, nil
}

func getWindowsV10Key(browserRoot string) ([]byte, error) {
	path := utils.FindMostRecentlyUsedFile(browserRoot, "Local State")
	if path == "" {
		return nil, errors.New("could not find Local State file")
	}
	log.Println("found local state file")
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open local state file: %v", err)
	}
	defer file.Close()
	data := make(map[string]interface{})
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, fmt.Errorf("could not decode local state file: %v", err)
	}
	base64Key, ok := data["os_crypt"].(map[string]interface{})["encrypted_key"].(string)
	if !ok {
		return nil, errors.New("could not find encrypted key in local state file")
	}
	encryptedKey, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("could not base64 decode encrypted key: %v", err)
	}
	prefix := []byte("DPAPI")
	if !bytes.HasPrefix(encryptedKey, prefix) {
		return nil, errors.New("encrypted key does not have DPAPI prefix")
	}
	return utils.DecryptWindowsDpapi(encryptedKey[len(prefix):])
}
