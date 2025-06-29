package lidario

import (
	"os"
	"path/filepath"
	"strings"
)

// isLazFile determines if a file is a LAZ file based on extension and magic bytes
func isLazFile(filename string) bool {
	// Quick check by file extension
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".laz" {
		return false
	}
	
	// Verify by reading magic bytes - LAZ files start with "LASF" like LAS files
	// but have compressed data after the header
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()
	
	// Read the first 4 bytes to check for LAS/LAZ signature
	signature := make([]byte, 4)
	n, err := file.Read(signature)
	if err != nil || n != 4 {
		return false
	}
	
	// Both LAS and LAZ files start with "LASF"
	return string(signature) == "LASF"
}

// isLasFile determines if a file is an uncompressed LAS file
func isLasFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".las" {
		return false
	}
	
	// Verify by reading magic bytes
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()
	
	signature := make([]byte, 4)
	n, err := file.Read(signature)
	if err != nil || n != 4 {
		return false
	}
	
	return string(signature) == "LASF"
}

// GetFileType returns the detected file type
func GetFileType(filename string) string {
	if isLazFile(filename) {
		return "LAZ"
	}
	if isLasFile(filename) {
		return "LAS"
	}
	return "UNKNOWN"
}