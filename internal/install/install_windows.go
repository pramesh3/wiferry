//go:build windows

package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const supported = true

type paths struct {
	dir, exe, icon, startMenu string
}

func winPaths() (paths, error) {
	local := os.Getenv("LOCALAPPDATA")
	roaming := os.Getenv("APPDATA")
	if local == "" || roaming == "" {
		return paths{}, fmt.Errorf("LOCALAPPDATA/APPDATA not set")
	}
	dir := filepath.Join(local, "Programs", "WiFerry")
	return paths{
		dir:       dir,
		exe:       filepath.Join(dir, "WiFerry.exe"),
		icon:      filepath.Join(dir, "WiFerry.ico"),
		startMenu: filepath.Join(roaming, "Microsoft", "Windows", "Start Menu", "Programs", "WiFerry.lnk"),
	}, nil
}

// powershell runs a script without flashing a console window.
func powershell(script string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func desktopShortcut() (string, error) {
	d, err := powershell("[Environment]::GetFolderPath('Desktop')")
	if err != nil || d == "" {
		return "", fmt.Errorf("find Desktop folder: %v %s", err, d)
	}
	return filepath.Join(d, "WiFerry.lnk"), nil
}

func makeShortcut(lnk, target, icon string) error {
	script := fmt.Sprintf(`$s = (New-Object -ComObject WScript.Shell).CreateShortcut(%s); `+
		`$s.TargetPath = %s; $s.WorkingDirectory = %s; $s.IconLocation = %s; `+
		`$s.Description = 'Send files between this PC and your phone over Wi-Fi'; $s.Save()`,
		psQuote(lnk), psQuote(target), psQuote(filepath.Dir(target)), psQuote(icon+",0"))
	if out, err := powershell(script); err != nil {
		return fmt.Errorf("create shortcut %s: %v %s", lnk, err, out)
	}
	return nil
}

func install(icons Icons) (Result, error) {
	p, err := winPaths()
	if err != nil {
		return Result{}, err
	}
	if err := copySelf(p.exe); err != nil {
		return Result{}, fmt.Errorf("copy app: %w", err)
	}
	if err := writeFile(p.icon, icons.ICO, 0o644); err != nil {
		return Result{}, fmt.Errorf("write icon: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(p.startMenu), 0o755); err != nil {
		return Result{}, err
	}
	if err := makeShortcut(p.startMenu, p.exe, p.icon); err != nil {
		return Result{}, err
	}
	res := Result{Exe: p.exe, Shortcuts: []string{p.startMenu}}
	desk, err := desktopShortcut()
	if err != nil {
		return res, err
	}
	if err := makeShortcut(desk, p.exe, p.icon); err != nil {
		return res, err
	}
	res.Shortcuts = append(res.Shortcuts, desk)
	return res, nil
}

func uninstall() error {
	p, err := winPaths()
	if err != nil {
		return err
	}
	files := []string{p.startMenu, p.icon}
	if d, err := desktopShortcut(); err == nil {
		files = append(files, d)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	// The exe can't delete itself while running; remove it only if this is
	// a different copy.
	if self, _ := os.Executable(); !strings.EqualFold(self, p.exe) {
		os.Remove(p.exe)
	}
	return nil
}

func installed() bool {
	p, err := winPaths()
	if err != nil {
		return false
	}
	_, err = os.Stat(p.startMenu)
	return err == nil
}
