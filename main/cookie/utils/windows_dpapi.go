//go:build windows

package utils

import (
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

func DecryptWindowsDpapi(ciphertext []byte) ([]byte, error) {
	var blobIn, blobOut windows.DataBlob
	blobIn.Size = uint32(len(ciphertext))
	blobIn.Data = &ciphertext[0]

	err := windows.CryptUnprotectData(&blobIn, nil, nil, 0, nil, 0, &blobOut)
	if err != nil {
		return nil, fmt.Errorf("CryptUnprotectData failed: %v", err)
	}

	d := make([]byte, blobOut.Size)
	copy(d, (*[1 << 30]byte)(unsafe.Pointer(blobOut.Data))[:])

	_, err = windows.LocalFree(windows.Handle(unsafe.Pointer(blobOut.Data)))
	if err != nil {
		return nil, fmt.Errorf("LocalFree failed: %v", err)
	}

	return d, nil
}
