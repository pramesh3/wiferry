// Command wiferry shares files between this PC and phones or other computers
// on the same Wi-Fi, with no internet and no app needed on the phone.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"wiferry/internal/auth"
	"wiferry/internal/netinfo"
	"wiferry/internal/qr"
	"wiferry/internal/server"
	"wiferry/internal/store"
)

const defaultPort = 8642

func main() {
	port := flag.Int("port", defaultPort, "port to listen on")
	dir := flag.String("dir", "", "folder for received and shared files (default ~/WiFerry)")
	noBrowser := flag.Bool("no-browser", false, "don't open the browser on start")
	flag.Parse()

	if err := run(*port, *dir, *noBrowser, isFlagSet("port")); err != nil {
		fmt.Fprintln(os.Stderr, "wiferry:", err)
		os.Exit(1)
	}
}

func isFlagSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) { set = set || f.Name == name })
	return set
}

func run(port int, dir string, noBrowser, portExplicit bool) error {
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

	h := server.New(server.Config{Auth: a, Store: st, Port: port, ListIPs: netinfo.List, SessionStart: sessionStart})
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 15 * time.Second}

	printBanner(a.Token(), port, dir)
	hostURL := fmt.Sprintf("http://127.0.0.1:%d/host", port)
	if !noBrowser {
		openBrowser(hostURL)
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
	fmt.Println("Press Ctrl+C to stop.")
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
