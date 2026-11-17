// Package qr renders connection URLs as QR codes.
package qr

import qrcode "github.com/skip2/go-qrcode"

// PNG returns a size×size PNG QR code for content.
func PNG(content string, size int) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, size)
}

// Terminal returns a QR code drawn with Unicode half blocks for printing.
func Terminal(content string) (string, error) {
	q, err := qrcode.New(content, qrcode.Low)
	if err != nil {
		return "", err
	}
	return q.ToSmallString(false), nil
}
