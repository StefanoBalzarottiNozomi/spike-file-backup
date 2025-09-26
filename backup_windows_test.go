package spikefilebackup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	junction "github.com/nyaosorg/go-windows-junction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCreateFile(t *testing.T) {
	// Test with real path
	realRootFolder := t.TempDir()
	realMidFolder := filepath.Join(realRootFolder, "real_mid_folder")
	mkdirErr1 := os.Mkdir(realMidFolder, os.ModePerm)
	require.NoError(t, mkdirErr1)
	realFinalFolder := filepath.Join(realMidFolder, "real_final_folder")
	mkdirErr2 := os.Mkdir(realFinalFolder, os.ModePerm)
	require.NoError(t, mkdirErr2)
	realFilePath := filepath.Join(realFinalFolder, "test.txt")
	realFd, realIoResult, realCreateErr := CreateFile(realFilePath)
	assert.NoError(t, realCreateErr)
	assert.NotNil(t, realIoResult)
	assert.NotEqual(t, 0, realFd)
	closeErr := windows.Close(realFd)
	require.NoError(t, closeErr)
	assert.FileExists(t, realFilePath)

	// Test with symlink path
	symlinkRootFolder := t.TempDir()
	symlinkMidFolder := filepath.Join(symlinkRootFolder, "symlink_mid_folder")
	symlinkErr := os.Symlink(realMidFolder, symlinkMidFolder)
	assert.NoError(t, symlinkErr)
	symlinkFinalFolder := filepath.Join(symlinkMidFolder, "symlink_final_folder")
	mkdirErr3 := os.Mkdir(symlinkFinalFolder, os.ModePerm)
	assert.NoError(t, mkdirErr3)
	symlinkFilePath := filepath.Join(symlinkFinalFolder, "test.txt")
	_, symLinkIoResult, symLinkCreateErr := CreateFile(symlinkFilePath)
	assert.Error(t, symLinkCreateErr)
	assert.NotNil(t, symLinkIoResult)
	assert.NoFileExists(t, symlinkFilePath)

	// Test with junction path
	junctionRootFolder := t.TempDir()
	junctionMidFolder := filepath.Join(junctionRootFolder, "junction_mid_folder")
	junctionErr := junction.Create(realMidFolder, junctionMidFolder)
	require.NoError(t, junctionErr)
	junctionFinalFolder := filepath.Join(junctionMidFolder, "junction_final_folder")
	mkdirErr4 := os.Mkdir(junctionFinalFolder, os.ModePerm)
	require.NoError(t, mkdirErr4)
	junctionFilePath := filepath.Join(junctionFinalFolder, "test.txt")
	_, junctionIoResult, junctionCreateErr := CreateFile(junctionFilePath)
	require.Error(t, junctionCreateErr)
	assert.NotNil(t, junctionIoResult)
	assert.NoFileExists(t, junctionFilePath)

	// Test with hardlink path
	hardlinkRootFolder := t.TempDir()
	hardlinkFinalFolder := filepath.Join(hardlinkRootFolder, "hardlink_final_folder")
	hardlinkErr2 := os.Mkdir(hardlinkFinalFolder, os.ModePerm)
	require.NoError(t, hardlinkErr2)
	hardlinkFilePath := filepath.Join(hardlinkFinalFolder, "test.txt")
	hardlinkErr := os.Link(realFilePath, hardlinkFilePath)
	require.NoError(t, hardlinkErr)
	_, hardlinkIoResult, hardlinkCreateErr := CreateFile(hardlinkFilePath)
	require.Error(t, hardlinkCreateErr)
	assert.NotNil(t, hardlinkIoResult)
}

func TestJunctionRaceAttack(t *testing.T) {
	rootFolder := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	const fileName = "file.txt"
	maliciousFolder := filepath.Join(rootFolder, "malicious_folder")
	vulnerableFolder := filepath.Join(rootFolder, "vulnerable_folder")
	fileInCorrectPath := filepath.Join(vulnerableFolder, fileName)
	fileInMaliciousPath := filepath.Join(maliciousFolder, fileName)

	assert.NoError(t, os.Mkdir(maliciousFolder, os.ModePerm))

	wg := sync.WaitGroup{}
	defer wg.Wait()
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_ = os.RemoveAll(vulnerableFolder)
				_ = os.Mkdir(vulnerableFolder, os.ModePerm)
				f, _ := os.Create(fileInCorrectPath)
				_ = f.Close()
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_ = os.RemoveAll(vulnerableFolder)
				_ = os.Mkdir(vulnerableFolder, os.ModePerm)
				_ = junction.Create(maliciousFolder, vulnerableFolder)
			}
		}
	}()
	before := time.Now()
	assert.Eventually(t, func() bool {
		_, statErr := os.Lstat(fileInMaliciousPath)
		return statErr == nil
	}, time.Minute*1, time.Millisecond*100)

	fmt.Printf("Malicious file detected after %d s", time.Since(before)/time.Second)
	cancel()
}

func TestJunctionRaceAttackFail(t *testing.T) {
	rootFolder := t.TempDir()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	const fileName = "file.txt"
	maliciousFolder := filepath.Join(rootFolder, "malicious_folder")
	vulnerableFolder := filepath.Join(rootFolder, "vulnerable_folder")
	fileInCorrectPath := filepath.Join(vulnerableFolder, fileName)
	fileInMaliciousPath := filepath.Join(maliciousFolder, fileName)

	assert.NoError(t, os.Mkdir(maliciousFolder, os.ModePerm))

	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_ = os.RemoveAll(vulnerableFolder)
				_ = os.Mkdir(vulnerableFolder, os.ModePerm)
				// fd, _ := os.Create(fileInCorrectPath)
				// _ = fd.Close()
				fd, _, _ := CreateFile(fileInCorrectPath)
				_ = windows.Close(fd)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_ = os.RemoveAll(vulnerableFolder)
				_ = os.Mkdir(vulnerableFolder, os.ModePerm)
				_ = junction.Create(maliciousFolder, vulnerableFolder)
			}
		}
	}()
	fileFound := atomic.Bool{}
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if _, statErr := os.Lstat(fileInMaliciousPath); statErr == nil {
					cancel()
					fileFound.Store(true)
					return
				}
			}
		}
	}()
	wg.Wait()
	assert.False(t, fileFound.Load(), "The file in the malicious path should not be found")
	cancel()
}
