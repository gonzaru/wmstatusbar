package util

import (
	"errors"
	"os"
	"strings"
)

// ReadMessageFile returns the content of the file
func ReadMessageFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
