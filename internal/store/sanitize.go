package store

import (
	"path"
	"strings"
)

const maxNameBytes = 200

// Characters not allowed in file names on Windows (and "/" everywhere).
const badChars = `<>:"/\|?*`

var reservedWindows = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// Sanitize turns an untrusted file name into a safe base name that is valid
// on Linux and Windows. It never returns an empty string, a path, or a
// hidden (dot-prefixed) name.
func Sanitize(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	name = path.Base(name)

	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(badChars, r) {
			b.WriteRune('_')
			continue
		}
		b.WriteRune(r)
	}
	name = strings.Trim(b.String(), " .")
	if name == "" {
		return "file"
	}

	stem := name
	if i := strings.IndexByte(name, '.'); i >= 0 {
		stem = name[:i]
	}
	if reservedWindows[strings.ToUpper(stem)] {
		name = "_" + name
	}

	if len(name) > maxNameBytes {
		ext := path.Ext(name)
		if len(ext) > 20 {
			ext = ""
		}
		runes := []rune(strings.TrimSuffix(name, ext))
		for len(string(runes))+len(ext) > maxNameBytes {
			runes = runes[:len(runes)-1]
		}
		name = string(runes) + ext
	}
	return name
}
