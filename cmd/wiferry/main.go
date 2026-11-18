// Command wiferry shares files between this PC and phones or other computers
// on the same Wi-Fi, with no internet and no app needed on the phone.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pramesh3/wiferry/internal/auth"
	"github.com/pramesh3/wiferry/internal/install"
	"github.com/pramesh3/wiferry/internal/netinfo"
	"github.com/pramesh3/wiferry/internal/qr"
	"github.com/pramesh3/wiferry/internal/server"
	"github.com/pramesh3/wiferry/internal/store"
	"github.com/pramesh3/wiferry/web"
)

const defaultPort = 8642

// version is set at build time: -ldflags "-X main.version=v0.1.1".
var version = "dev"

func main() {
	port := flag.Int("port", defaultPort, "port to listen on")
	dir := flag.String("dir", "", "folder for received and shared files (default ~/WiFerry)")
	noBrowser := flag.Bool("no-browser", false, "don't open the browser on start")
	doInstall := flag.Bool("install", false, "add WiFerry to the app menu and desktop, then exit")
	doUninstall := flag.Bool("uninstall", false, "remove WiFerry's shortcuts and installed app (keeps your files), then exit")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("wiferry", version)
		return
	}

	var err error
	switch {
	case *doInstall:
		err = runInstall()
	case *doUninstall:
		if err = install.Uninstall(); err == nil {
			fmt.Println("WiFerry shortcuts removed. Your files in ~/WiFerry were kept.")
		}
	default:
		err = run(*port, *dir, *noBrowser, isFlagSet("port"))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "wiferry:", err)
		os.Exit(1)
	}
}

func isFlagSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) { set = set || f.Name == name })
	return set
}

func icons() install.Icons {
	png, _ := fs.ReadFile(web.FS, "icon.png")
	ico, _ := fs.ReadFile(web.FS, "icon.ico")
	return install.Icons{PNG: png, ICO: ico}
}

func runInstall() error {
	if !install.Supported() {
		return errors.New("shortcuts are not supported on this system yet")
	}
	res, err := install.Install(icons())
	if err != nil {
		return err
	}
	fmt.Println("WiFerry installed:", res.Exe)
	for _, s := range res.Shortcuts {
		fmt.Println("  shortcut:", s)
	}
	return nil
}

// alreadyRunning reports whether a WiFerry is already serving on port.
func alreadyRunning(port int) bool {
	c := http.Client{Timeout: time.Second}
	resp, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	return string(b) == server.HealthBody
}

func run(port int, dir string, noBrowser, portExplicit bool) error {
	// Launching again (e.g. clicking the desktop icon twice) just reopens
	// the running instance instead of starting a second one.
	if alreadyRunning(port) {
		fmt.Println("WiFerry is already running; opening it.")
		if !noBrowser {
			openBrowser(fmt.Sprintf("http://127.0.0.1:%d/host", port))
		}
		return nil
	}

	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir = filepath.Join(home, "WiFerry")
	}
	st, err := store.New(dir)
	if err != nil {
		return fmt.Errorf("create folders: %w", err)
	}
	// Each run is a fresh session: Shared only holds copies, so start empty.
	if err := st.Clear(store.Shared); err != nil {
		return fmt.Errorf("clear shared folder: %w", err)
	}
	sessionStart := time.Now()

	a, err := auth.New()
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		if portExplicit {
			return fmt.Errorf("port %d: %w", port, err)
		}
		if ln, err = net.Listen("tcp", ":0"); err != nil {
			return err
		}
	}
	port = ln.Addr().(*net.TCPAddr).Port

	srv := &http.Server{ReadHeaderTimeout: 15 * time.Second}
	cfg := server.Config{
		Auth: a, Store: st, Port: port, ListIPs: netinfo.List, SessionStart: sessionStart, Version: version,
		OnQuit: func() {
			log.Println("stopped from the WiFerry page")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			srv.Shutdown(ctx)
		},
	}
	if install.Supported() {
		cfg.Install = func() (install.Result, error) { return install.Install(icons()) }
		cfg.Installed = install.Installed
	}
	srv.Handler = server.New(cfg)

	printBanner(a.Token(), port, dir)
	if !noBrowser {
		openBrowser(fmt.Sprintf("http://127.0.0.1:%d/host", port))
	}

	err = srv.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func printBanner(token string, port int, dir string) {
	fmt.Printf("\nWiFerry is running. Files are saved in %s\n", dir)
	fmt.Printf("On this PC open: http://127.0.0.1:%d/host\n\n", port)
	ips, _ := netinfo.List()
	if len(ips) == 0 {
		fmt.Println("No Wi-Fi/LAN address found. Connect to Wi-Fi or a hotspot and restart.")
		return
	}
	for _, ip := range ips {
		fmt.Printf("  %-16s http://%s:%d/?t=%s\n", ip.Iface, ip.IP, port, token)
	}
	if s, err := qr.Terminal(fmt.Sprintf("http://%s:%d/?t=%s", ips[0].IP, port, token)); err == nil {
		fmt.Printf("\nScan with your phone camera (%s):\n%s\n", ips[0].IP, s)
	}
	fmt.Println("Press Ctrl+C or use Stop on the WiFerry page to quit.")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("could not open browser: %v (open %s manually)", err, url)
		return
	}
	go cmd.Wait()
}
