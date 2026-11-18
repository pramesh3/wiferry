package main

// Smoke test of the real binary: build it, run it, and do a full phone↔PC
// hand-off over the LAN address. Runs on every OS (CI covers Linux and
// Windows). Skipped with -short.

import (
	"bufio"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBinarySmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the binary")
	}
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "wiferry")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-ldflags", "-X main.version=smoke", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	out, err := exec.Command(bin, "--version").Output()
	if err != nil || strings.TrimSpace(string(out)) != "wiferry smoke" {
		t.Fatalf("--version = %q, %v", out, err)
	}

	dir := filepath.Join(tmp, "data")
	cmd := exec.Command(bin, "--no-browser", "--dir", dir, "--port", "18950")
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })

	// The banner prints the phone URL (LAN IP + token).
	urlRe := regexp.MustCompile(`http://(\d+\.\d+\.\d+\.\d+):18950/\?t=[A-Za-z0-9_-]+`)
	found := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if m := urlRe.FindString(sc.Text()); m != "" {
				found <- m
				break
			}
		}
		io.Copy(io.Discard, stdout)
	}()
	var phoneURL string
	select {
	case phoneURL = <-found:
	case <-time.After(15 * time.Second):
		t.Skip("no LAN address on this machine; nothing for a phone to connect to")
	}
	base := phoneURL[:strings.Index(phoneURL, "/?t=")]
	host := "http://127.0.0.1:18950"

	jar, _ := cookiejar.New(nil)
	phone := &http.Client{Jar: jar, Timeout: 10 * time.Second}
	pc := &http.Client{Timeout: 10 * time.Second}

	// expect runs a request and checks its status; returns the body.
	expect := func(what string, want int, do func() (*http.Response, error)) []byte {
		t.Helper()
		resp, err := do()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s: status %d, want %d: %s", what, resp.StatusCode, want, b)
		}
		return b
	}
	upload := func(c *http.Client, url, name string, data []byte) (*http.Response, error) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", name)
		fw.Write(data)
		mw.Close()
		req, _ := http.NewRequest("POST", url, &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("X-WiFerry", "1")
		return c.Do(req)
	}
	post := func(c *http.Client, url string) (*http.Response, error) {
		req, _ := http.NewRequest("POST", url, nil)
		req.Header.Set("X-WiFerry", "1")
		return c.Do(req)
	}

	expect("LAN request without token", 403, func() (*http.Response, error) { return http.Get(base + "/api/files?box=shared") })
	expect("phone opens QR URL", 200, func() (*http.Response, error) { return phone.Get(phoneURL) })

	// Phone → PC, then the PC saves it and it's removed.
	photo := bytes.Repeat([]byte("photo-bytes "), 200000)
	expect("phone upload", 200, func() (*http.Response, error) { return upload(phone, base+"/api/upload", "IMG_0001.HEIC", photo) })
	got := expect("PC download", 200, func() (*http.Response, error) { return pc.Get(host + "/api/download/received/IMG_0001.HEIC") })
	if !bytes.Equal(got, photo) {
		t.Fatal("PC download differs from upload")
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(dir, "Received", "IMG_0001.HEIC")); !os.IsNotExist(err) {
		t.Fatalf("received file not removed after PC download (err=%v)", err)
	}

	// PC → phone, then the phone saves it and it's removed.
	doc := []byte("boarding pass")
	expect("PC share", 200, func() (*http.Response, error) { return upload(pc, host+"/api/share", "pass.pdf", doc) })
	got = expect("phone download", 200, func() (*http.Response, error) { return phone.Get(base + "/api/download/shared/pass.pdf") })
	if !bytes.Equal(got, doc) {
		t.Fatal("phone download differs")
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(dir, "Shared", "pass.pdf")); !os.IsNotExist(err) {
		t.Fatalf("shared file not removed after phone download (err=%v)", err)
	}

	// Unsafe names are made safe on this OS.
	expect("unsafe name upload", 200, func() (*http.Response, error) { return upload(phone, base+"/api/upload", `..\..\CON.txt`, []byte("x")) })
	if _, err := os.Stat(filepath.Join(dir, "Received", "_CON.txt")); err != nil {
		t.Fatalf("reserved name not sanitized: %v", err)
	}

	// Second launch reopens the running one instead of starting another.
	second, err := exec.Command(bin, "--no-browser", "--dir", dir, "--port", "18950").CombinedOutput()
	if err != nil || !strings.Contains(string(second), "already running") {
		t.Fatalf("second launch: %v %s", err, second)
	}

	// Stop from the PC page ends the process.
	expect("Stop", 204, func() (*http.Response, error) { return post(pc, host+"/api/quit") })
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("process still running after Stop")
	}
}
