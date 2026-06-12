//go:build !windows

package ai

import "errors"

func protect([]byte) ([]byte, error) {
	return nil, errors.New("DPAPI is only available on Windows")
}

func unprotect([]byte) ([]byte, error) {
	return nil, errors.New("DPAPI is only available on Windows")
}
