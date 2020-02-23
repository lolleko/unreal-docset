package main

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Untar file
func Untar(dst string, tarPath string, db *sql.DB) error {

	tarFile, err := os.Open(tarPath)
	defer tarFile.Close()

	if err != nil {
		return err
	}

	gzr, err := gzip.NewReader(tarFile)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}

		// if it's a file create it
		case tar.TypeReg:
			err := os.MkdirAll(filepath.Dir(target), 0755)
			if err != nil {
				fmt.Println(target)
				return err
			}
			f, err := os.Create(target)
			if err != nil {
				return err
			}

			if filepath.Ext(target) == ".html" {
				transformedHTML, entryName, entryType, err := transformHTML(target, tr)
				if err != nil {
					return err
				}
				f.WriteString(transformedHTML)
				entryPath, _ := filepath.Rel(dst, target)
				addEntryToDatabase(db, entryName, entryType, entryPath)
			} else {
				if _, err := io.Copy(f, tr); err != nil {
					return err
				}
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
