package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"wiferry/internal/auth"
	"wiferry/internal/netinfo"
	"wiferry/internal/store"
)

const token = "tok123"

type fixture struct {
	h     http.Handler
	store *store.Store
}

func setup(t *testing.T) *fixture {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h := New(Config{
		Auth:  auth.NewWithToken(token),
		Store: st,
		Port:  8642,
		ListIPs: func() ([]netinfo.Addr, error) {
			return []netinfo.Addr{{IP: "192.168.1.10", Iface: "wlan0"}}, nil
		},
	})
	return &fixture{h: h, store: st}
}

func (f *fixture) do(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func hostReq(method, url string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, url, body)
	r.RemoteAddr = "127.0.0.1:5000"
	r.Host = "127.0.0.1:8642"
	return r
}

func clientReq(method, url string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, url, body)
	r.RemoteAddr = "192.168.1.50:5000"
	r.Host = "192.168.1.10:8642"
	r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	return r
}

func multipartFile(t *testing.T, name, content string) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	io.WriteString(fw, content)
	mw.Close()
	return &buf, mw.FormDataContentType()
}

func upload(t *testing.T, f *fixture, mk func(string, string, io.Reader) *http.Request, url, name, content string, csrf bool) *httptest.ResponseRecorder {
	t.Helper()
	body, ct := multipartFile(t, name, content)
	r := mk("POST", url, body)
	r.Header.Set("Content-Type", ct)
	if csrf {
		r.Header.Set("X-WiFerry", "1")
	}
	return f.do(r)
}

func TestClientUploadToReceived(t *testing.T) {
	f := setup(t)
	rec := upload(t, f, clientReq, "/api/upload", "IMG_0001.HEIC", "photo-bytes", true)
	if rec.Code != 200 {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body)
	}
	var resp struct{ Saved []store.FileInfo }
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Saved) != 1 || resp.Saved[0].Name != "IMG_0001.HEIC" {
		t.Fatalf("resp = %s", rec.Body)
	}
	b, err := os.ReadFile(filepath.Join(f.store.Dir(store.Received), "IMG_0001.HEIC"))
	if err != nil || string(b) != "photo-bytes" {
		t.Fatalf("file = %q err=%v", b, err)
	}
}

func TestUploadWithoutCSRFHeaderRejected(t *testing.T) {
	f := setup(t)
	if rec := upload(t, f, clientReq, "/api/upload", "a.txt", "x", false); rec.Code != 403 {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestUnauthenticatedClientRejected(t *testing.T) {
	f := setup(t)
	r := httptest.NewRequest("GET", "/api/files?box=shared", nil)
	r.RemoteAddr = "192.168.1.50:5000"
	if rec := f.do(r); rec.Code != 403 {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestReceivedIsHostOnly(t *testing.T) {
	f := setup(t)
	if rec := f.do(clientReq("GET", "/api/files?box=received", nil)); rec.Code != 403 {
		t.Fatalf("client code = %d", rec.Code)
	}
	if rec := f.do(hostReq("GET", "/api/files?box=received", nil)); rec.Code != 200 {
		t.Fatalf("host code = %d", rec.Code)
	}
}

func TestHostSharesClientDownloads(t *testing.T) {
	f := setup(t)
	if rec := upload(t, f, hostReq, "/api/share", "doc.pdf", "PDFDATA", true); rec.Code != 200 {
		t.Fatalf("share code = %d %s", rec.Code, rec.Body)
	}
	if rec := upload(t, f, clientReq, "/api/share", "x.pdf", "x", true); rec.Code != 403 {
		t.Fatalf("client share code = %d", rec.Code)
	}

	rec := f.do(clientReq("GET", "/api/files?box=shared", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "doc.pdf") {
		t.Fatalf("list = %d %s", rec.Code, rec.Body)
	}

	rec = f.do(clientReq("GET", "/api/download/shared/doc.pdf", nil))
	if rec.Code != 200 || rec.Body.String() != "PDFDATA" {
		t.Fatalf("download = %d %q", rec.Code, rec.Body)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment") {
		t.Fatalf("content-disposition = %q", cd)
	}
}

func sharedNames(f *fixture) []string {
	list, _ := f.store.List(store.Shared)
	names := []string{}
	for _, fi := range list {
		names = append(names, fi.Name)
	}
	return names
}

func TestClientDownloadDeletesSharedFile(t *testing.T) {
	f := setup(t)
	f.store.Save(store.Shared, "a.jpg", strings.NewReader("AAAA"))
	f.store.Save(store.Shared, "b.jpg", strings.NewReader("BBBB"))
	rec := f.do(clientReq("GET", "/api/download/shared/a.jpg", nil))
	if rec.Code != 200 || rec.Body.String() != "AAAA" {
		t.Fatalf("download = %d %q", rec.Code, rec.Body)
	}
	if got := sharedNames(f); len(got) != 1 || got[0] != "b.jpg" {
		t.Fatalf("shared after download = %v", got)
	}
}

func TestDownloadsThatKeepFile(t *testing.T) {
	cases := map[string]func() *http.Request{
		"host download": func() *http.Request { return hostReq("GET", "/api/download/shared/a.jpg", nil) },
		"partial range": func() *http.Request {
			r := clientReq("GET", "/api/download/shared/a.jpg", nil)
			r.Header.Set("Range", "bytes=0-1")
			return r
		},
		"HEAD": func() *http.Request { return clientReq("HEAD", "/api/download/shared/a.jpg", nil) },
	}
	for name, mk := range cases {
		f := setup(t)
		f.store.Save(store.Shared, "a.jpg", strings.NewReader("AAAA"))
		f.do(mk())
		if got := sharedNames(f); len(got) != 1 {
			t.Errorf("%s: shared = %v, want file kept", name, got)
		}
	}
}

func TestClientZipDeletesSharedFiles(t *testing.T) {
	f := setup(t)
	f.store.Save(store.Shared, "a.jpg", strings.NewReader("AAAA"))
	f.store.Save(store.Shared, "b.jpg", strings.NewReader("BBBB"))
	rec := f.do(clientReq("GET", "/api/zip/shared", nil))
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil || len(zr.File) != 2 {
		t.Fatalf("zip err=%v", err)
	}
	if got := sharedNames(f); len(got) != 0 {
		t.Fatalf("shared after zip = %v", got)
	}
}

func TestHostZipKeepsFiles(t *testing.T) {
	f := setup(t)
	f.store.Save(store.Shared, "a.jpg", strings.NewReader("AAAA"))
	f.do(hostReq("GET", "/api/zip/shared", nil))
	if got := sharedNames(f); len(got) != 1 {
		t.Fatalf("shared after host zip = %v", got)
	}
}

func TestZipNameHasDateStamp(t *testing.T) {
	f := setup(t)
	rec := f.do(clientReq("GET", "/api/zip/shared", nil))
	cd := rec.Header().Get("Content-Disposition")
	if !regexp.MustCompile(`^attachment; filename=WiFerry-\d{4}-\d{2}-\d{2}_\d{2}-\d{2}\.zip$`).MatchString(cd) {
		t.Fatalf("content-disposition = %q", cd)
	}
}

func TestDownloadMissingIs404(t *testing.T) {
	f := setup(t)
	if rec := f.do(clientReq("GET", "/api/download/shared/nope.txt", nil)); rec.Code != 404 {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestHostPageHostOnly(t *testing.T) {
	f := setup(t)
	if rec := f.do(clientReq("GET", "/host", nil)); rec.Code != 403 {
		t.Fatalf("client code = %d", rec.Code)
	}
	rec := f.do(hostReq("GET", "/host", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "WiFerry") {
		t.Fatalf("host code = %d", rec.Code)
	}
	rec = f.do(clientReq("GET", "/", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "WiFerry") {
		t.Fatalf("client page code = %d", rec.Code)
	}
	if rec := f.do(clientReq("GET", "/static/client.js", nil)); rec.Code != 200 {
		t.Fatalf("static code = %d", rec.Code)
	}
}

func TestHostDelete(t *testing.T) {
	f := setup(t)
	f.store.Save(store.Shared, "a.txt", strings.NewReader("x"))
	r := clientReq("DELETE", "/api/files/shared/a.txt", nil)
	r.Header.Set("X-WiFerry", "1")
	if rec := f.do(r); rec.Code != 403 {
		t.Fatalf("client delete code = %d", rec.Code)
	}
	r = hostReq("DELETE", "/api/files/shared/a.txt", nil)
	r.Header.Set("X-WiFerry", "1")
	if rec := f.do(r); rec.Code != 204 {
		t.Fatalf("host delete code = %d", rec.Code)
	}
	if list, _ := f.store.List(store.Shared); len(list) != 0 {
		t.Fatal("file not deleted")
	}
}

func TestInfoAndQR(t *testing.T) {
	f := setup(t)
	rec := f.do(hostReq("GET", "/api/info", nil))
	var info struct {
		URLs []struct{ IP, URL string }
	}
	json.Unmarshal(rec.Body.Bytes(), &info)
	if rec.Code != 200 || len(info.URLs) != 1 || info.URLs[0].URL != "http://192.168.1.10:8642/?t="+token {
		t.Fatalf("info = %d %s", rec.Code, rec.Body)
	}
	if rec := f.do(clientReq("GET", "/api/info", nil)); rec.Code != 403 {
		t.Fatalf("client info code = %d", rec.Code)
	}
	rec = f.do(hostReq("GET", "/api/qr.png?ip=192.168.1.10", nil))
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("qr code = %d", rec.Code)
	}
	if rec := f.do(hostReq("GET", "/api/qr.png?ip=8.8.8.8", nil)); rec.Code != 400 {
		t.Fatalf("bad ip code = %d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	f := setup(t)
	r := httptest.NewRequest("GET", "/healthz", nil)
	if rec := f.do(r); rec.Code != 200 {
		t.Fatalf("code = %d", rec.Code)
	}
}
