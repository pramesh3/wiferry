//go:build linux

package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const supported = true

type paths struct {
	exe, icon, menuEntry, desktopEntry string
}

func linuxPaths() (paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return paths{}, err
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home, ".local", "share")
	}
	p := paths{
		exe:       filepath.Join(home, ".local", "bin", "wiferry"),
		icon:      filepath.Join(data, "icons", "hicolor", "256x256", "apps", "wiferry.png"),
		menuEntry: filepath.Join(data, "applications", "wiferry.desktop"),
	}
	if d := desktopDir(home); d != "" {
		p.desktopEntry = filepath.Join(d, "wiferry.desktop")
	}
	return p, nil
}

// desktopDir finds the user's Desktop folder (localized names included).
func desktopDir(home string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "xdg-user-dir", "DESKTOP").Output(); err == nil {
		d := strings.TrimSpace(string(out))
		if d != "" && d != home {
			if fi, err := os.Stat(d); err == nil && fi.IsDir() {
				return d
			}
		}
	}
	d := filepath.Join(home, "Desktop")
	if fi, err := os.Stat(d); err == nil && fi.IsDir() {
		return d
	}
	return ""
}

func desktopFile(exe string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=WiFerry
GenericName=File Transfer
Comment=Send photos and files between this PC and your phone over Wi-Fi
Exec="%s"
Icon=wiferry
Terminal=false
Categories=Network;FileTransfer;Utility;
Keywords=phone;iphone;transfer;wifi;share;photos;
StartupNotify=false
`, exe)
}

func install(icons Icons) (Result, error) {
	p, err := linuxPaths()
	if err != nil {
		return Result{}, err
	}
	if err := copySelf(p.exe); err != nil {
		return Result{}, fmt.Errorf("copy app: %w", err)
	}
	if err := writeFile(p.icon, icons.PNG, 0o644); err != nil {
		return Result{}, fmt.Errorf("write icon: %w", err)
	}
	entry := []byte(desktopFile(p.exe))
	if err := writeFile(p.menuEntry, entry, 0o755); err != nil {
		return Result{}, fmt.Errorf("app menu entry: %w", err)
	}
	res := Result{Exe: p.exe, Shortcuts: []string{p.menuEntry}}
	if p.desktopEntry != "" {
		if err := writeFile(p.desktopEntry, entry, 0o755); err != nil {
			return res, fmt.Errorf("desktop shortcut: %w", err)
		}
		trust(p.desktopEntry)
		res.Shortcuts = append(res.Shortcuts, p.desktopEntry)
	}
	return res, nil
}

// trust marks a desktop launcher as allowed to run on GNOME, so users don't
// have to right-click → "Allow Launching". Best effort.
func trust(path string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	exec.CommandContext(ctx, "gio", "set", path, "metadata::trusted", "true").Run()
}

func uninstall() error {
	p, err := linuxPaths()
	if err != nil {
		return err
	}
	for _, f := range []string{p.menuEntry, p.desktopEntry, p.icon, p.exe} {
		if f == "" {
			continue
		}
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func installed() bool {
	p, err := linuxPaths()
	if err != nil {
		return false
	}
	_, err = os.Stat(p.menuEntry)
	return err == nil
}
