package spikefilebackup

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ntDll                          = syscall.NewLazyDLL("ntdll.dll")
	procRtlDosPathNameToNtPathName = ntDll.NewProc("RtlDosPathNameToNtPathName_U")
	procRtlFreeUnicodeString       = ntDll.NewProc("RtlFreeUnicodeString")
)

func StringToUnicodeString(s string) (*windows.NTUnicodeString, error) {
	utf16Ptr, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil, err
	}

	utf16Slice := (*[1 << 20]uint16)(unsafe.Pointer(utf16Ptr))
	length := 0
	for utf16Slice[length] != 0 {
		length++
	}

	return &windows.NTUnicodeString{
		Length:        uint16(length * 2),       // Length in bytes
		MaximumLength: uint16((length + 1) * 2), // Include null terminator
		Buffer:        utf16Ptr,
	}, nil
}

func UnicodeStringToString(us *windows.NTUnicodeString) string {
	if us.Buffer == nil || us.Length == 0 {
		return ""
	}
	return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(us.Buffer))[:us.Length/2])
}

func ConvertPathToNtPath(path string) (string, error) {
	pathUtf16Ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	unicodeString := windows.NTUnicodeString{}

	ntStatus, _, _ := procRtlDosPathNameToNtPathName.Call(
		uintptr(unsafe.Pointer(pathUtf16Ptr)),
		uintptr(unsafe.Pointer(&unicodeString)),
		0, 0,
	)
	if ntStatus != 1 {
		return "", fmt.Errorf("failed to convert path to NT path: %w", syscall.Errno(ntStatus))
	}
	defer procRtlFreeUnicodeString.Call(uintptr(unsafe.Pointer(&unicodeString)))

	ntPath := UnicodeStringToString(&unicodeString)
	return ntPath, nil
}

func CreateFile(path string) (windows.Handle, windows.IO_STATUS_BLOCK, error) {
	ioResult := windows.IO_STATUS_BLOCK{}
	ntPath, err := ConvertPathToNtPath(path)
	if err != nil {
		return 0, ioResult, err
	}
	unicodeObjectName, err2 := StringToUnicodeString(ntPath)
	if err2 != nil {
		return 0, ioResult, err2
	}

	// windows.OBJ_DONT_REPARSE  Ensure we don't follow path symlinks or reparse pints
	objectAttributes := windows.OBJECT_ATTRIBUTES{
		ObjectName: unicodeObjectName,
		Attributes: windows.OBJ_DONT_REPARSE,
		Length:     uint32(unsafe.Sizeof(windows.OBJECT_ATTRIBUTES{})),
	}
	var fileHandle windows.Handle

	err3 := windows.NtCreateFile(
		&fileHandle,
		windows.GENERIC_ALL,
		&objectAttributes,
		&ioResult,
		nil,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
		windows.FILE_CREATE,
		// FILE_OPEN_REPARSE_POINT probably is useless if we don't open an existing file
		windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT,
		0,
		0)

	return fileHandle, ioResult, err3
}
