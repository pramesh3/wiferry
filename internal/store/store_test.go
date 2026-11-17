package store

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"photo.jpg":        "photo.jpg",
		"../../etc/passwd": "passwd",
		`a\b.jpg`:          "b.jpg",
		"..":               "file",
		"":                 "file",
		"CON.txt":          "_CON.txt",
		"nul":              "_nul",
		"a<b>.png":         "a_b_.png",
		".bashrc":          "bashrc",
		"tab\there.txt":    "tab_here.txt",
		"日本 写真.HEIC":       "日本 写真.HEIC",
		"trailing. ":       "trailing",
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
	long := strings.Repeat("é", 300) + ".jpg"
	got := Sanitize(long)
	if len(got) > 200 || !strings.HasSuffix(got, ".jpg") {
		t.Errorf("long name: len=%d suffix ok=%v", len(got), strings.HasSuffix(got, ".jpg"))
	}
}

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewCreatesBoxes(t *testing.T) {
	s := newStore(t)
	for _, b := range []Box{Received, Shared} {
		if fi, err := os.Stat(s.Dir(b)); err != nil || !fi.IsDir() {
			t.Fatalf("box %s missing: %v", b, err)
		}
	}
}

func TestParseBox(t *testing.T) {
	if b, ok := ParseBox("shared"); !ok || b != Shared {
		t.Fatal("shared")
	}
	if _, ok := ParseBox("../x"); ok {
		t.Fatal("bad box accepted")
	}
}

func TestSaveDuplicateNames(t *testing.T) {
	s := newStore(t)
	names := []string{}
	for i := 0; i < 3; i++ {
		fi, err := s.Save(Received, "img.jpg", strings.NewReader("x"))
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, fi.Name)
	}
	want := []string{"img.jpg", "img (1).jpg", "img (2).jpg"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %v, want %v", names, want)
		}
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestSaveFailureLeavesNothing(t *testing.T) {
	s := newStore(t)
	if _, err := s.Save(Received, "a.txt", io.MultiReader(strings.NewReader("part"), failReader{})); err == nil {
		t.Fatal("expected error")
	}
	entries, _ := os.ReadDir(s.Dir(Received))
	if len(entries) != 0 {
		t.Fatalf("leftover files: %v", entries)
	}
}

func TestListHidesDotfilesAndDirs(t *testing.T) {
	s := newStore(t)
	s.Save(Shared, "a.txt", strings.NewReader("hello"))
	os.WriteFile(filepath.Join(s.Dir(Shared), ".hidden"), []byte("x"), 0o644)
	os.Mkdir(filepath.Join(s.Dir(Shared), "sub"), 0o755)
	list, err := s.List(Shared)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "a.txt" || list[0].Size != 5 {
		t.Fatalf("list = %+v", list)
	}
}

func TestOpenAndDelete(t *testing.T) {
	s := newStore(t)
	s.Save(Shared, "a.txt", strings.NewReader("hello"))
	f, _, err := s.Open(Shared, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	f.Close()
	if string(b) != "hello" {
		t.Fatalf("content = %q", b)
	}
	if _, _, err := s.Open(Shared, "../Received/a.txt"); err == nil {
		t.Fatal("traversal allowed")
	}
	if err := s.Delete(Shared, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Open(Shared, "a.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("after delete err = %v", err)
	}
}

func TestWriteZip(t *testing.T) {
	s := newStore(t)
	s.Save(Shared, "a.txt", strings.NewReader("AAA"))
	s.Save(Shared, "b.txt", strings.NewReader("BB"))
	var buf bytes.Buffer
	if err := s.WriteZip(Shared, &buf); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(b)
	}
	if got["a.txt"] != "AAA" || got["b.txt"] != "BB" || len(got) != 2 {
		t.Fatalf("zip = %v", got)
	}
}
