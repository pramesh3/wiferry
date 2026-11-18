//go:build windows

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Creates real Start Menu and Desktop shortcuts on a Windows machine (CI),
// checks they point at the installed exe, then removes them.
func TestInstallUninstallWindows(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", filepath.Join(tmp, "Local"))
	t.Setenv("APPDATA", filepath.Join(tmp, "Roaming"))

	if Installed() {
		t.Fatal("installed before install")
	}
	res, err := Install(Icons{ICO: []byte("ico")})
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(tmp, "Local", "Programs", "WiFerry", "WiFerry.exe")
	if !strings.EqualFold(res.Exe, exe) {
		t.Fatalf("exe = %s", res.Exe)
	}
	if _, err := os.Stat(exe); err != nil {
		t.Fatalf("exe not copied: %v", err)
	}
	if len(res.Shortcuts) != 2 || !Installed() {
		t.Fatalf("shortcuts = %v installed=%v", res.Shortcuts, Installed())
	}
	for _, lnk := range res.Shortcuts {
		target, err := powershell("(New-Object -ComObject WScript.Shell).CreateShortcut(" + psQuote(lnk) + ").TargetPath")
		if err != nil {
			t.Fatalf("%s: read target: %v", lnk, err)
		}
		// Compare files, not strings: Windows may report the same folder
		// by its short (RUNNER~1) or long (runneradmin) name.
		a, errA := os.Stat(target)
		b, errB := os.Stat(exe)
		if errA != nil || errB != nil || !os.SameFile(a, b) {
			t.Fatalf("%s target = %q, want %s", lnk, target, exe)
		}
	}

	if err := Uninstall(); err != nil {
		t.Fatal(err)
	}
	for _, f := range append(res.Shortcuts, filepath.Join(tmp, "Local", "Programs", "WiFerry", "WiFerry.ico")) {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Fatalf("%s not removed", f)
		}
	}
	if Installed() {
		t.Fatal("still installed")
	}
}
