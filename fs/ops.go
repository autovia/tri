package fs

import (
	"os"
	"path/filepath"
)

func CleanupEmptyDirs(path string, root string) {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}
	for {
		if dir == root {
			break
		}
		if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
			os.Remove(dir)
		}
		dir = filepath.Dir(dir)
	}
}
