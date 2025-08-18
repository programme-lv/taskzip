package zips

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Unzip extracts a zip archive to a specified destination.
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Prevent Zip Slip by ensuring the file path is within the destination directory
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			// Create directory with appropriate permissions
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// Ensure the parent directory exists
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// Create the destination file
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		// It's better to use defer to ensure files are closed even if io.Copy fails
		defer outFile.Close()

		// Open the ZIP file entry
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// Copy the file content
		if _, err := io.Copy(outFile, rc); err != nil {
			return err
		}
	}
	return nil
}

// ZipDir zips the contents of srcDir into dstZipFile path. The resulting
// archive will contain the directory contents with a top-level folder name
// equal to the base of srcDir.
func ZipDir(srcDir string, dstZipFile string) error {
	// ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(dstZipFile), os.ModePerm); err != nil {
		return err
	}

	zipFile, err := os.Create(dstZipFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	base := filepath.Base(srcDir)

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		zipPath := filepath.ToSlash(filepath.Join(base, relPath))

		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			// add a directory entry (with trailing slash)
			_, err := zw.CreateHeader(&zip.FileHeader{
				Name:   strings.TrimSuffix(zipPath, "/") + "/",
				Method: zip.Deflate,
			})
			return err
		}

		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		fh.Name = zipPath
		fh.Method = zip.Deflate

		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	}

	return filepath.WalkDir(srcDir, walkFn)
}
