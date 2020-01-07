package utils

import (
	"io/ioutil"
	"os"
)

// TempDir extends the functionality of ioutil.TempDir by adding a pathOnly argument. If pathOnly
// is true, then TempDir returns a path to a non-existent directory.
func TempDir(dir, prefix string, pathOnly bool) (string, error) {
	tempDir, err := ioutil.TempDir(dir, prefix)
	if err != nil {
		return "", err
	}

	if pathOnly {
		err = os.RemoveAll(tempDir)
		if err != nil {
			return "", err
		}
	}

	return tempDir, nil
}
