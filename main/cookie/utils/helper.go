package utils

import (
	"log"
	"os"
	"path/filepath"
)

func FindMostRecentlyUsedFile(root, filename string) string {
	i := 0
	paths := make([]string, 0)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		i++
		if info != nil && info.Name() == filename {
			paths = append(paths, path)
		}
		return nil
	})

	log.Println("searched", i, "files")

	if len(paths) == 0 {
		return ""
	}

	maxPath := paths[0]
	info, _ := os.Lstat(maxPath)
	maxModTime := info.ModTime()
	for _, path := range paths[1:] {
		info, _ = os.Lstat(maxPath)
		modTime := info.ModTime()
		if modTime.After(maxModTime) {
			maxPath = path
			maxModTime = modTime
		}
	}
	return maxPath
}

func ConfigHome() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return xdgConfigHome
	}
	return filepath.Join(os.Getenv("HOME"), ".config")
}
