// Package server wires WiFerry's HTTP routes.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"wiferry/internal/auth"
	"wiferry/internal/netinfo"
	"wiferry/internal/qr"
	"wiferry/internal/store"
	"wiferry/web"
)

// Config holds the server's dependencies.
type Config struct {
	Auth    *auth.Auth
	Store   *store.Store
	Port    int
	ListIPs func() ([]netinfo.Addr, error)
	// SessionStart hides Received files from earlier runs (they stay on disk).
	SessionStart time.Time
}

type srv struct {
	Config

	// receipts maps an opaque id (given to the sending device) to the file it
	// uploaded, so the device can see when the PC has picked the file up
	// without being able to list or probe the Received folder.
	mu       sync.Mutex
	receipts map[string]string
}

// New returns the complete WiFerry HTTP handler.
func New(cfg Config) http.Handler {
	if cfg.ListIPs == nil {
		cfg.ListIPs = netinfo.List
	}
	s := &srv{Config: cfg, receipts: map[string]string{}}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") })
	mux.HandleFunc("GET /{$}", s.page("client.html"))
	mux.HandleFunc("GET /host", hostOnly(s.page("host.html")))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticOnly(http.FileServerFS(web.FS))))

	mux.HandleFunc("GET /api/info", hostOnly(s.info))
	mux.HandleFunc("GET /api/qr.png", hostOnly(s.qrPNG))
	mux.HandleFunc("POST /api/upload", s.receive(store.Received))
	mux.HandleFunc("POST /api/share", hostOnly(s.receive(store.Shared)))
	mux.HandleFunc("GET /api/files", s.list)
	mux.HandleFunc("GET /api/sent", s.sent)
	mux.HandleFunc("GET /api/download/{box}/{name}", s.download)
	mux.HandleFunc("GET /api/zip/{box}", s.zip)
	mux.HandleFunc("DELETE /api/files/{box}/{name}", hostOnly(s.delete))

	return cfg.Auth.Middleware(csrf(mux))
}

// csrf requires a custom header on state-changing API calls. Browsers can't
// send it cross-origin without a CORS preflight, which we never approve.
func csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Header.Get("X-WiFerry") != "1" {
			http.Error(w, "missing X-WiFerry header", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hostOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsLoopback(r) {
			http.Error(w, "only available on the PC running WiFerry", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

func staticOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path.Ext(r.URL.Path) {
		case ".css", ".js", ".svg":
			h.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func (s *srv) page(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(web.FS, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

// canRead: anyone authenticated may read Shared; only the host reads Received.
func canRead(r *http.Request, b store.Box) bool {
	return b == store.Shared || auth.IsLoopback(r)
}

func (s *srv) box(w http.ResponseWriter, r *http.Request, v string) (store.Box, bool) {
	b, ok := store.ParseBox(v)
	if !ok {
		http.Error(w, "unknown box", http.StatusBadRequest)
		return "", false
	}
	if !canRead(r, b) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return "", false
	}
	return b, true
}

func (s *srv) url(ip string) string {
	return fmt.Sprintf("http://%s:%d/?t=%s", ip, s.Port, s.Auth.Token())
}

type urlInfo struct {
	IP    string `json:"ip"`
	Iface string `json:"iface"`
	URL   string `json:"url"`
}

func (s *srv) info(w http.ResponseWriter, r *http.Request) {
	ips, err := s.ListIPs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	urls := make([]urlInfo, 0, len(ips))
	for _, a := range ips {
		urls = append(urls, urlInfo{IP: a.IP, Iface: a.Iface, URL: s.url(a.IP)})
	}
	host, _ := os.Hostname()
	writeJSON(w, map[string]any{"hostname": host, "dir": s.Store.Base(), "port": s.Port, "urls": urls})
}

func (s *srv) qrPNG(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	ips, _ := s.ListIPs()
	known := false
	for _, a := range ips {
		known = known || a.IP == ip
	}
	if !known {
		http.Error(w, "unknown ip", http.StatusBadRequest)
		return
	}
	png, err := qr.PNG(s.url(ip), 320)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(png)
}

func (s *srv) receive(b store.Box) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			http.Error(w, "expected multipart upload", http.StatusBadRequest)
			return
		}
		type savedItem struct {
			store.FileInfo
			ID string `json:"id,omitempty"`
		}
		saved := []savedItem{}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, "upload interrupted", http.StatusBadRequest)
				return
			}
			if part.FileName() == "" {
				part.Close()
				continue
			}
			fi, err := s.Store.Save(b, part.FileName(), part)
			part.Close()
			if err != nil {
				log.Printf("save %q: %v", part.FileName(), err)
				if errors.Is(err, syscall.ENOSPC) {
					http.Error(w, "Not enough space on the PC", http.StatusInsufficientStorage)
				} else {
					http.Error(w, "upload failed: "+err.Error(), http.StatusInternalServerError)
				}
				return
			}
			log.Printf("received %s (%d bytes) from %s", fi.Name, fi.Size, r.RemoteAddr)
			item := savedItem{FileInfo: fi}
			if b == store.Received {
				item.ID = s.addReceipt(fi.Name)
			}
			saved = append(saved, item)
		}
		writeJSON(w, map[string]any{"saved": saved})
	}
}

func (s *srv) addReceipt(name string) string {
	buf := make([]byte, 12)
	rand.Read(buf)
	id := hex.EncodeToString(buf)
	s.mu.Lock()
	s.receipts[id] = name
	s.mu.Unlock()
	return id
}

// sent reports which of the caller's receipt ids still wait in Received.
// Unknown ids, and files already downloaded or deleted, are not pending.
func (s *srv) sent(w http.ResponseWriter, r *http.Request) {
	pending := []string{}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
		name, ok := s.receipts[id]
		if !ok {
			continue
		}
		f, _, err := s.Store.Open(store.Received, name)
		if err != nil {
			delete(s.receipts, id)
			continue
		}
		f.Close()
		pending = append(pending, id)
	}
	writeJSON(w, map[string]any{"pending": pending})
}

func (s *srv) list(w http.ResponseWriter, r *http.Request) {
	b, ok := s.box(w, r, r.URL.Query().Get("box"))
	if !ok {
		return
	}
	files, err := s.files(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, files)
}

// files lists a box as shown to users: Received only includes this session.
func (s *srv) files(b store.Box) ([]store.FileInfo, error) {
	all, err := s.Store.List(b)
	if err != nil || b != store.Received {
		return all, err
	}
	out := []store.FileInfo{}
	for _, fi := range all {
		if !fi.Modified.Before(s.SessionStart) {
			out = append(out, fi)
		}
	}
	return out, nil
}

// attachment marks a response as a file download. For other devices (the
// phone) the type is generic binary: iOS Safari otherwise opens zips and
// other known types in a preview screen instead of saving them to Files.
func attachment(w http.ResponseWriter, r *http.Request, name string) {
	if !auth.IsLoopback(r) {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
}

func (s *srv) download(w http.ResponseWriter, r *http.Request) {
	b, ok := s.box(w, r, r.PathValue("box"))
	if !ok {
		return
	}
	name := r.PathValue("name")
	f, fi, err := s.Store.Open(b, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	attachment(w, r, name)
	cw := &countingWriter{ResponseWriter: w, status: http.StatusOK}
	http.ServeContent(cw, r, name, fi.ModTime(), f)
	f.Close() // must close before delete (Windows)

	if consumes(r, b) && r.Method == http.MethodGet && cw.status == http.StatusOK && cw.n == fi.Size() {
		s.removeDelivered(b, []string{name}, r)
	}
}

// consumes reports whether a completed download should remove the file
// because it has reached the other side: Shared files taken by another
// device, or Received files saved by this PC's browser.
func consumes(r *http.Request, b store.Box) bool {
	if auth.IsLoopback(r) {
		return b == store.Received
	}
	return b == store.Shared
}

func (s *srv) removeDelivered(b store.Box, names []string, r *http.Request) {
	for _, n := range names {
		if err := s.Store.Delete(b, n); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("remove delivered %s: %v", n, err)
			continue
		}
		log.Printf("delivered %s to %s, removed from Shared", n, r.RemoteAddr)
	}
}

// countingWriter records the status code and body bytes written so we can
// tell a complete download from a partial, cached or failed one.
type countingWriter struct {
	http.ResponseWriter
	status int
	n      int64
}

func (c *countingWriter) WriteHeader(code int) {
	c.status = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.ResponseWriter.Write(p)
	c.n += int64(n)
	return n, err
}

func (s *srv) zip(w http.ResponseWriter, r *http.Request) {
	b, ok := s.box(w, r, r.PathValue("box"))
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	// Dated name: unzipping on iPhone/Windows/macOS creates a folder with this
	// name, and repeated downloads never overwrite each other.
	attachment(w, r, "WiFerry-"+time.Now().Format("2006-01-02_15-04")+".zip")
	files, err := s.files(b)
	if err == nil {
		err = s.Store.WriteZip(b, files, w)
	}
	if err != nil {
		log.Printf("zip %s: %v", b, err) // incomplete: keep files
		return
	}
	if consumes(r, b) {
		names := make([]string, len(files))
		for i, fi := range files {
			names[i] = fi.Name
		}
		s.removeDelivered(b, names, r)
	}
}

func (s *srv) delete(w http.ResponseWriter, r *http.Request) {
	b, ok := s.box(w, r, r.PathValue("box"))
	if !ok {
		return
	}
	if err := s.Store.Delete(b, r.PathValue("name")); err != nil {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
