// Package store keeps received and shared files on disk.
package store

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Box is one of the two file areas.
type Box string

const (
	// Received holds files sent to this PC by other devices.
	Received Box = "received"
	// Shared holds files this PC offers to other devices.
	Shared Box = "shared"
)

var boxDirs = map[Box]string{Received: "Received", Shared: "Shared"}

// ParseBox converts a URL value into a Box.
func ParseBox(s string) (Box, bool) {
	b := Box(s)
	_, ok := boxDirs[b]
	return b, ok
}

// FileInfo describes a stored file.
type FileInfo struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// Store manages files under a base directory.
type Store struct {
	base string
	mu   sync.Mutex // serializes choosing a unique name + rename
}

// New creates the base directory and box directories if missing.
func New(base string) (*Store, error) {
	s := &Store{base: base}
	for b := range boxDirs {
		if err := os.MkdirAll(s.Dir(b), 0o755); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Base returns the base directory.
func (s *Store) Base() string { return s.base }

// Dir returns the directory for a box.
func (s *Store) Dir(b Box) string { return filepath.Join(s.base, boxDirs[b]) }

// Save streams r into box b under a sanitized, unique version of name.
// Partial data is never left behind on error.
func (s *Store) Save(b Box, name string, r io.Reader) (FileInfo, error) {
	dir := s.Dir(b)
	tmp, err := os.CreateTemp(dir, ".wiferry-*.part")
	if err != nil {
		return FileInfo{}, err
	}
	tmpName := tmp.Name()
	n, err := io.Copy(tmp, r)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmpName)
		return FileInfo{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	final := uniqueName(dir, Sanitize(name))
	if err := os.Rename(tmpName, filepath.Join(dir, final)); err != nil {
		os.Remove(tmpName)
		return FileInfo{}, err
	}
	return FileInfo{Name: final, Size: n, Modified: time.Now()}, nil
}

func uniqueName(dir, name string) string {
	if _, err := os.Lstat(filepath.Join(dir, name)); os.IsNotExist(err) {
		return name
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		c := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		if _, err := os.Lstat(filepath.Join(dir, c)); os.IsNotExist(err) {
			return c
		}
	}
}

// List returns regular, non-hidden files in a box, newest first.
func (s *Store) List(b Box) ([]FileInfo, error) {
	entries, err := os.ReadDir(s.Dir(b))
	if err != nil {
		return nil, err
	}
	out := []FileInfo{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || !e.Type().IsRegular() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, FileInfo{Name: e.Name(), Size: info.Size(), Modified: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out, nil
}

func (s *Store) path(b Box, name string) (string, error) {
	if name == "" || Sanitize(name) != name {
		return "", os.ErrNotExist
	}
	return filepath.Join(s.Dir(b), name), nil
}

// Open opens a file in a box for reading.
func (s *Store) Open(b Box, name string) (*os.File, os.FileInfo, error) {
	p, err := s.path(b, name)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		f.Close()
		return nil, nil, os.ErrNotExist
	}
	return f, fi, nil
}

// Delete removes a file from a box.
func (s *Store) Delete(b Box, name string) error {
	p, err := s.path(b, name)
	if err != nil {
		return err
	}
	return os.Remove(p)
}

// WriteZip streams all files in a box as an uncompressed zip archive
// (photos and videos are already compressed). It returns the names written
// so callers can act on exactly those files once the transfer succeeds.
func (s *Store) WriteZip(b Box, w io.Writer) ([]string, error) {
	files, err := s.List(b)
	if err != nil {
		return nil, err
	}
	zw := zip.NewWriter(w)
	names := make([]string, 0, len(files))
	for _, fi := range files {
		if err := addToZip(zw, filepath.Join(s.Dir(b), fi.Name), fi); err != nil {
			return nil, err
		}
		names = append(names, fi.Name)
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return names, nil
}

func addToZip(zw *zip.Writer, p string, fi FileInfo) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	w, err := zw.CreateHeader(&zip.FileHeader{Name: fi.Name, Method: zip.Store, Modified: fi.Modified})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}
