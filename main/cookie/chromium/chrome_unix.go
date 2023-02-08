//go:build linux

package chromium

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"errors"
	"github.com/elvis972602/kemono-scraper/main/cookie/utils/linux"
	"golang.org/x/crypto/pbkdf2"
	"log"
)

const (
	salt   = "saltysalt"
	secret = "peanuts"
)

var (
	// 16 bytes
	iv = []byte("                ")
)

type ChromeCookieDecryptor struct {
	v10Key []byte
	v11Key []byte
	v10    int
	v11    int
	other  int
}

func NewChromeCookieDecryptor(browserRoot, keyringName string, keyring linux.LinuxKeyring) (*ChromeCookieDecryptor, error) {
	l := &ChromeCookieDecryptor{
		v10:   0,
		v11:   0,
		other: 0,
	}
	l.v10Key = pbkdf2Sha1([]byte(secret), []byte(salt), 1, 16)
	password, err := getLinuxKeyringPassword(keyringName, keyring)
	if err != nil {
		return nil, err
	}
	if password != nil {
		l.v11Key = pbkdf2Sha1(password, []byte(salt), 1, 16)
	}
	return l, nil
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

		return aesCBCDecrypt(ciphertext, d.v10Key, iv)
	} else {
		d.v11++
		if d.v11Key == nil {
			return nil, errors.New("v11 key is nil")
		}
		return aesCBCDecrypt(ciphertext, d.v11Key, iv)
	}
}

func getLinuxKeyringPassword(keyringName string, keyring linux.LinuxKeyring) ([]byte, error) {
	if !linux.LinuxKeyringNames[keyring] {
		keyring = linux.ChooseLinuxKeyring()
	}
	log.Println("Using keyring:", keyring)
	if keyring == linux.KWALLET {
		return linux.GetKWalletPassword(keyringName)
	} else if keyring == linux.GNOMEKEYRING {
		return linux.GetGnomeKeyringPassword(keyringName)
	} else if keyring == linux.BASICTEXT {
		// all store as v10
		return nil, nil
	}
	return nil, errors.New("unknown keyring")
}

func pbkdf2Sha1(password, salt []byte, iterations, keyLen int) []byte {
	return pbkdf2.Key(password, salt, iterations, keyLen, sha1.New)
}

func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkcs7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func aesCBCEncrypt(plaintext []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	plaintext = pkcs7Padding(plaintext, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(plaintext))
	blockMode.CryptBlocks(crypted, plaintext)
	return crypted, nil
}

func aesCBCDecrypt(ciphertext []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, iv[:blockSize])
	origData := make([]byte, len(ciphertext))
	blockMode.CryptBlocks(origData, ciphertext)
	origData = pkcs7UnPadding(origData)
	return origData, nil
}
