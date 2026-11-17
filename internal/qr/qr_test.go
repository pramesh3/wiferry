package qr

import (
	"bytes"
	"testing"
)

func TestPNG(t *testing.T) {
	b, err := PNG("http://192.168.1.2:8642/?t=abc", 256)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("\x89PNG")) {
		t.Fatal("not a PNG")
	}
}

func TestTerminal(t *testing.T) {
	s, err := Terminal("http://192.168.1.2:8642/?t=abc")
	if err != nil || len(s) < 100 {
		t.Fatalf("len=%d err=%v", len(s), err)
	}
}
