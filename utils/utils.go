package utils

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)


var ValidExtensions = []string{

	".3gp", ".3gpp", ".avi", ".flv", ".insv", ".m2t", ".m2ts", ".m4v",
	".mkv", ".mov", ".mp4", ".mpe", ".mpeg", ".mpg", ".mts", ".vob",
	".webm", ".wmv",


	".3fr", ".ari", ".arw", ".cap", ".cin", ".cr2", ".cr3", ".crw",
	".dcr", ".dng", ".erf", ".fff", ".iiq", ".k25", ".kdc", ".mrw",
	".nef", ".nrw", ".orf", ".ori", ".pef", ".psd", ".raf", ".raw",
	".rw2", ".rwl", ".sr2", ".srf", ".srw", ".x3f", ".avif", ".bmp",
	".gif", ".heic", ".heif", ".hif", ".insp", ".jp2", ".jpe", ".jpeg",
	".jpg", ".jxl", ".png", ".svg", ".tif", ".tiff", ".webp",
}


func IsValidExtension(filename string) bool {

	ext := strings.ToLower(filepath.Ext(filename))
	for _, validExt := range ValidExtensions {
		if ext == validExt {
			return true
		}
	}

	return false
}


func CalculateChecksum(filePath string) (string, error) {
	
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("Error opening the file: %v", err)
	}
	defer file.Close()


	hash := sha1.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", fmt.Errorf("Error calculating hash: %v", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
