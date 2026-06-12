package ai

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func protect(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, nil
	}
	in := windows.DataBlob{Size: uint32(len(plain)), Data: &plain[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return blobBytes(out), nil
}

func unprotect(cipher []byte) ([]byte, error) {
	if len(cipher) == 0 {
		return nil, nil
	}
	in := windows.DataBlob{Size: uint32(len(cipher)), Data: &cipher[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return blobBytes(out), nil
}

func blobBytes(b windows.DataBlob) []byte {
	if b.Size == 0 || b.Data == nil {
		return nil
	}
	src := unsafe.Slice(b.Data, int(b.Size))
	out := make([]byte, len(src))
	copy(out, src)
	return out
}
