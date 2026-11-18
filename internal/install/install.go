// Package install puts WiFerry in the user's app menu and on the desktop, so
// people can launch it like any other app. It only touches the current
// user's folders and never needs admin rights.
package install

import (
	"io"
	"os"
	"path/filepath"
)

// Icons are the image files written next to the installed app.
type Icons struct {
	PNG []byte // 256×256, used on Linux
	ICO []byte // multi-size, used on Windows
}

// Result says where things were put.
type Result struct {
	Exe       string   `json:"exe"`
	Shortcuts []string `json:"shortcuts"`
}

// Supported reports whether shortcuts can be created on this OS.
func Supported() bool { return supported }

// Install copies the running binary to a per-user location and creates
// app-menu and desktop shortcuts pointing at it.
func Install(icons Icons) (Result, error) { return install(icons) }

// Uninstall removes the shortcuts, icon and installed binary. User files in
// ~/WiFerry are kept.
func Uninstall() error { return uninstall() }

// Installed reports whether the app-menu shortcut exists.
func Installed() bool { return installed() }

// copySelf copies the running executable to dst unless it already runs
// from there. It writes to a temp file first so a failure leaves no
// half-written binary.
func copySelf(dst string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if s, err := filepath.EvalSymlinks(self); err == nil {
		self = s
	}
	if d, err := filepath.EvalSymlinks(dst); err == nil && d == self {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(self)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".wiferry-install-*")
	if err != nil {
		return err
	}
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}
