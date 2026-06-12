package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procOpenFileMapping = modkernel32.NewProc("OpenFileMappingW")
	procOpenMutex       = modkernel32.NewProc("OpenMutexW")
)

func openFileMapping(access uint32, inheritHandle bool, name *uint16) (windows.Handle, error) {
	inherit := uintptr(0)
	if inheritHandle {
		inherit = 1
	}
	r0, _, e1 := syscall.SyscallN(procOpenFileMapping.Addr(), uintptr(access), inherit, uintptr(unsafe.Pointer(name)))
	if r0 == 0 {
		if e1 != 0 {
			return 0, e1
		}
		return 0, syscall.EINVAL
	}
	return windows.Handle(r0), nil
}

func openMutex() (windows.Handle, func()) {
	name, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		return 0, nil
	}
	h, err := openMutexHandle(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return 0, nil
	}
	wait, err := windows.WaitForSingleObject(h, 1000)
	if err != nil || wait != windows.WAIT_OBJECT_0 {
		return h, nil
	}
	return h, func() { _ = windows.ReleaseMutex(h) }
}

func openMutexHandle(access uint32, inheritHandle bool, name *uint16) (windows.Handle, error) {
	inherit := uintptr(0)
	if inheritHandle {
		inherit = 1
	}
	r0, _, e1 := syscall.SyscallN(procOpenMutex.Addr(), uintptr(access), inherit, uintptr(unsafe.Pointer(name)))
	if r0 == 0 {
		if e1 != 0 {
			return 0, e1
		}
		return 0, syscall.EINVAL
	}
	return windows.Handle(r0), nil
}
