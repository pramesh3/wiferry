//go:build linux

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallUninstallLinux(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	os.Mkdir(filepath.Join(home, "Desktop"), 0o755)

	if Installed() {
		t.Fatal("installed before install")
	}
	res, err := Install(Icons{PNG: []byte("png")})
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(home, ".local", "bin", "wiferry")
	if res.Exe != exe {
		t.Fatalf("exe = %s", res.Exe)
	}
	if fi, err := os.Stat(exe); err != nil || fi.Mode()&0o100 == 0 {
		t.Fatalf("exe not copied/executable: %v", err)
	}
	icon := filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps", "wiferry.png")
	if b, _ := os.ReadFile(icon); string(b) != "png" {
		t.Fatal("icon missing")
	}
	menu := filepath.Join(home, ".local", "share", "applications", "wiferry.desktop")
	desk := filepath.Join(home, "Desktop", "wiferry.desktop")
	for _, f := range []string{menu, desk} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if !strings.Contains(string(b), `Exec="`+exe+`"`) || !strings.Contains(string(b), "Icon=wiferry") {
			t.Fatalf("bad entry:\n%s", b)
		}
	}
	if len(res.Shortcuts) != 2 || !Installed() {
		t.Fatalf("shortcuts = %v installed=%v", res.Shortcuts, Installed())
	}

	// Installing again (e.g. from the installed copy) is fine.
	if _, err := Install(Icons{PNG: []byte("png")}); err != nil {
		t.Fatal(err)
	}

	if err := Uninstall(); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{menu, desk, icon, exe} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Fatalf("%s not removed", f)
		}
	}
	if Installed() {
		t.Fatal("still installed")
	}
}

func TestNoDesktopFolderStillInstallsMenuEntry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	res, err := Install(Icons{PNG: []byte("png")})
	if err != nil || len(res.Shortcuts) != 1 {
		t.Fatalf("res=%v err=%v", res, err)
	}
}
