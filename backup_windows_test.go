package spikefilebackup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert"
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
	realRootFolder := t.TempDir()
	realMidFolder := filepath.Join(realRootFolder, "real_mid_folder")
	mkdirErr1 := os.Mkdir(realMidFolder, os.ModePerm)
	assert.NoError(t, mkdirErr1)
	realFinalFolder := filepath.Join(realMidFolder, "real_final_folder")
	mkdirErr2 := os.Mkdir(realFinalFolder, os.ModePerm)
	assert.NoError(t, mkdirErr2)
	realFilePath := filepath.Join(realFinalFolder, "test.txt")
	fd, ioResult, createErr := CreateFile(realFilePath)
	assert.NoError(t, createErr)
	assert.Equal(t, ioResult.Status, windows.STATUS_SUCCESS)
	assert.NotEqual(t, 0, fd)
	closeErr := windows.CloseHandle(fd)
	assert.NoError(t, closeErr)
	AssertFileExists(t, realFilePath)
}
