package service

import (
	"io"
	"os"
	"path/filepath"

	"github.com/alexmullins/zip"
)

func UnzipWithPassword(zipPath, dest, password string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		f.SetPassword(password)
		fPath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fPath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fPath), 0755); err != nil {
			return nil
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()
		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}
	return nil
}
