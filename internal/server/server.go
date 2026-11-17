// Package server wires WiFerry's HTTP routes.
package server

import (
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
}

type srv struct{ Config }

// New returns the complete WiFerry HTTP handler.
func New(cfg Config) http.Handler {
	if cfg.ListIPs == nil {
		cfg.ListIPs = netinfo.List
	}
	s := &srv{cfg}
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
		saved := []store.FileInfo{}
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
			saved = append(saved, fi)
		}
		writeJSON(w, map[string]any{"saved": saved})
	}
}

func (s *srv) list(w http.ResponseWriter, r *http.Request) {
	b, ok := s.box(w, r, r.URL.Query().Get("box"))
	if !ok {
		return
	}
	files, err := s.Store.List(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, files)
}

func attachment(w http.ResponseWriter, name string) {
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
	defer f.Close()
	attachment(w, name)
	http.ServeContent(w, r, name, fi.ModTime(), f)
}

func (s *srv) zip(w http.ResponseWriter, r *http.Request) {
	b, ok := s.box(w, r, r.PathValue("box"))
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	// Dated name: unzipping on iPhone/Windows/macOS creates a folder with this
	// name, and repeated downloads never overwrite each other.
	attachment(w, "WiFerry-"+time.Now().Format("2006-01-02_15-04")+".zip")
	if err := s.Store.WriteZip(b, w); err != nil {
		log.Printf("zip %s: %v", b, err)
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
