package collector

// Thin PDH (Performance Data Helper) wrapper. PdhAddEnglishCounterW is used
// deliberately: counter paths stay in English even on localized Windows
// (Chinese fleet machines), where Get-Counter with English paths would fail.

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modpdh                          = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQueryW               = modpdh.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounterW       = modpdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData         = modpdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterValue = modpdh.NewProc("PdhGetFormattedCounterValue")
	procPdhCloseQuery               = modpdh.NewProc("PdhCloseQuery")
)

const pdhFmtDouble = 0x00000200

type pdhFmtCounterValue struct {
	CStatus uint32
	_       uint32
	Double  float64
}

type pdhQuery struct {
	handle   uintptr
	counters map[string]uintptr
}

func openPdhQuery() (*pdhQuery, error) {
	var h uintptr
	r, _, _ := procPdhOpenQueryW.Call(0, 0, uintptr(unsafe.Pointer(&h)))
	if r != 0 {
		return nil, fmt.Errorf("PdhOpenQuery failed: 0x%x", r)
	}
	return &pdhQuery{handle: h, counters: map[string]uintptr{}}, nil
}

// addCounter registers an English counter path under a short key. Returns an
// error without failing the whole query, so optional counters can degrade.
func (q *pdhQuery) addCounter(key, englishPath string) error {
	p, err := windows.UTF16PtrFromString(englishPath)
	if err != nil {
		return err
	}
	var h uintptr
	r, _, _ := procPdhAddEnglishCounterW.Call(q.handle, uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&h)))
	if r != 0 {
		return fmt.Errorf("PdhAddEnglishCounter(%s) failed: 0x%x", englishPath, r)
	}
	q.counters[key] = h
	return nil
}

// collect gathers one round of data. The first call after opening only primes
// rate counters; values are valid from the second call on.
func (q *pdhQuery) collect() error {
	r, _, _ := procPdhCollectQueryData.Call(q.handle)
	if r != 0 {
		return fmt.Errorf("PdhCollectQueryData failed: 0x%x", r)
	}
	return nil
}

// value returns the formatted double for a registered counter key; ok=false
// when the counter is missing or its current value is invalid.
func (q *pdhQuery) value(key string) (float64, bool) {
	h, exists := q.counters[key]
	if !exists {
		return 0, false
	}
	var v pdhFmtCounterValue
	r, _, _ := procPdhGetFormattedCounterValue.Call(h, pdhFmtDouble, 0, uintptr(unsafe.Pointer(&v)))
	if r != 0 {
		return 0, false
	}
	return v.Double, true
}

func (q *pdhQuery) close() {
	_, _, _ = procPdhCloseQuery.Call(q.handle)
}
