package migration

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func datafilesBackup() (string, error) {

	tmpDir, err := os.MkdirTemp("", "datafiles_backup_*")
	if err != nil {
		return "", err
	}

	c := configs.GetConfig()
	datafilesFolder := string(c.FilePaths.DataFiles)

	err = copyDir(datafilesFolder, tmpDir)
	if err != nil {
		return "", err
	}

	return tmpDir, nil
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			_, err := os.Stat(destPath)
			if errors.Is(err, os.ErrNotExist) {
				return os.MkdirAll(destPath, 0755)
			}
			return nil
		}

		// It’s a file
		return copyFile(path, destPath)
	})
}

// CopyFile copies a single file
func copyFile(srcFile, dstFile string) error {
	srcF, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer srcF.Close()

	dstF, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer dstF.Close()

	_, err = io.Copy(dstF, srcF)
	return err
}
