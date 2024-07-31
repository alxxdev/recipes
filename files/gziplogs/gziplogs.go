package main

import (
	"compress/gzip"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"time"
)

func gzCompress(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	w := gzip.NewWriter(out)
	defer w.Close()

	// Update metadata, must be before io.Copy
	w.Name = src
	info, err := file.Stat()
	if err == nil {
		w.ModTime = info.ModTime()
	}

	if _, err := io.Copy(w, file); err != nil {
		os.Remove(dest)
		return err
	}

	return nil
}

// SHA1 > 2025
func fileSHA1(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", nil
	}
	defer file.Close()

	var r io.Reader = file
	if path.Ext(fileName) == ".gz" {
		var err error
		r, err = gzip.NewReader(r)
		if err != nil {
			return "", err
		}
	}

	w := sha1.New()
	if _, err := io.Copy(w, r); err != nil {
		return "", err
	}

	sig := fmt.Sprintf("%x", w.Sum(nil))
	return sig, nil
}

func sameSig(file1, file2 string) (bool, error) {
	sig1, err := fileSHA1(file1)
	if err != nil {
		return false, err
	}

	sig2, err := fileSHA1(file2)
	if err != nil {
		return false, err
	}

	return sig1 == sig2, nil
}

func shouldCompress(path string, maxAge time.Duration) bool {
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("warning: %q: can't get info: %s", path, err)
		return false
	}

	if info.IsDir() {
		return false
	}

	return time.Since(info.ModTime()) >= maxAge
}

func filesToCompress(dir string, maxAge time.Duration) ([]string, error) {
	root := os.DirFS(dir)
	logFiles, err := fs.Glob(root, "*.log")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, src := range logFiles {
		name := path.Join(dir, src)
		if shouldCompress(name, maxAge) {
			names = append(names, name)
		}
	}

	return names, nil
}

func compressFiles(rootDir string, maxAge time.Duration) error {
	files, err := filesToCompress(rootDir, maxAge)
	if err != nil {
		return err
	}

	for _, src := range files {
		dest := src + ".gz"
		if err := gzCompress(src, dest); err != nil {
			return fmt.Errorf("%q: %w", src, err)
		}

		match, err := sameSig(src, dest)
		if err != nil {
			return err
		}

		if !match {
			return fmt.Errorf("%q <-> %q: signature don't match", src, dest)
		}

		if err := os.Remove(src); err != nil {
			log.Printf("warning: %q: can't delete - %s", src, err)
		}
	}

	return nil
}

func main() {
	var rootDir string
	flag.StringVar(&rootDir, "root", ".", "root dir")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: gzlogs")
		flag.PrintDefaults()
	}
	flag.Parse()

	const maxAge = 30 * 24 * time.Hour
	if err := compressFiles(rootDir, maxAge); err != nil {
		log.Fatalf("error: %s", err)
	}
}
