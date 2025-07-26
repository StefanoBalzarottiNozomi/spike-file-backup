package spikefilebackup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert"
	junction "github.com/nyaosorg/go-windows-junction"
	"golang.org/x/sys/windows"
)

func TestStringConversion(t *testing.T) {
	original := "Hello, World!"
	unicodeString, err := StringToUnicodeString(original)
	assert.NoError(t, err)
	assert.Equal(t, original, UnicodeStringToString(unicodeString))

	emptyUnicodeString, emptyErr := StringToUnicodeString("")
	assert.NoError(t, emptyErr)
	assert.Equal(t, "", UnicodeStringToString(emptyUnicodeString))
}

func TestConvertPathToNtPath(t *testing.T) {
	originalPath := `C:\Users\Public\Documents`
	expectedNtPath := `\??\C:\Users\Public\Documents`

	ntPath, err := ConvertPathToNtPath(originalPath)
	assert.NoError(t, err)
	assert.Equal(t, expectedNtPath, ntPath)
}

func AssertFileExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("Expected file to exist at %s, but it does not", path)
	}
	assert.NoError(t, err, "File should exist")
}

func TestCreateFile(t *testing.T) {
	// Test with real path
	realRootFolder := t.TempDir()
	realMidFolder := filepath.Join(realRootFolder, "real_mid_folder")
	mkdirErr1 := os.Mkdir(realMidFolder, os.ModePerm)
	assert.NoError(t, mkdirErr1)
	realFinalFolder := filepath.Join(realMidFolder, "real_final_folder")
	mkdirErr2 := os.Mkdir(realFinalFolder, os.ModePerm)
	assert.NoError(t, mkdirErr2)
	realFilePath := filepath.Join(realFinalFolder, "test.txt")
	realFd, realIoResult, realCreateErr := CreateFile(realFilePath)
	assert.NoError(t, realCreateErr)
	assert.NotNil(t, realIoResult)
	assert.NotEqual(t, 0, realFd)
	closeErr := windows.CloseHandle(realFd)
	assert.NoError(t, closeErr)
	AssertFileExists(t, realFilePath)

	// Test with symlink path
	symlinkRootFolder := t.TempDir()
	symlinkMidFolder := filepath.Join(symlinkRootFolder, "symlink_mid_folder")
	symlinkErr := os.Symlink(realMidFolder, symlinkMidFolder)
	assert.NoError(t, symlinkErr)
	symlinkFinalFolder := filepath.Join(symlinkMidFolder, "symlink_final_folder")
	mkdirErr3 := os.Mkdir(symlinkFinalFolder, os.ModePerm)
	assert.NoError(t, mkdirErr3)
	symlinkFilePath := filepath.Join(symlinkFinalFolder, "test.txt")
	_, symLinlIoResult, realCreateErr := CreateFile(symlinkFilePath)
	assert.Error(t, realCreateErr)
	assert.NotNil(t, symLinlIoResult)

	// Test with junction path
	junctionRootFolder := t.TempDir()
	junctionMidFolder := filepath.Join(junctionRootFolder, "junction_mid_folder")
	junctionErr := junction.Create(realMidFolder, junctionMidFolder)
	assert.NoError(t, junctionErr)
	junctionFinalFolder := filepath.Join(junctionMidFolder, "junction_final_folder")
	mkdirErr4 := os.Mkdir(junctionFinalFolder, os.ModePerm)
	assert.NoError(t, mkdirErr4)
	junctionFilePath := filepath.Join(junctionFinalFolder, "test.txt")
	_, junctionIoResult, junctionCreateErr := CreateFile(junctionFilePath)
	assert.Error(t, junctionCreateErr)
	assert.NotNil(t, junctionIoResult)

	// Test with hardlink path
	hardlinkRootFolder := t.TempDir()
	hardlinkFinalFolder := filepath.Join(hardlinkRootFolder, "hardlink_final_folder")
	hardlinkErr2 := os.Mkdir(hardlinkFinalFolder, os.ModePerm)
	assert.NoError(t, hardlinkErr2)
	hardlinkFilePath := filepath.Join(hardlinkFinalFolder, "test.txt")
	hardlinkErr := os.Link(realFilePath, hardlinkFilePath)
	assert.NoError(t, hardlinkErr)
	_, hardlinkIoResult, hardlinkCreateErr := CreateFile(hardlinkFilePath)
	assert.Error(t, hardlinkCreateErr)
	assert.NotNil(t, hardlinkIoResult)
}
