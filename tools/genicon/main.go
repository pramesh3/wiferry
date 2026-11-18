// Command genicon draws the WiFerry app icon and writes the PNG/ICO files
// embedded in web/. Run from the repo root: go run ./tools/genicon
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

func main() {
	write("web/icon.png", encodePNG(draw(256)))
	write("web/apple-touch-icon.png", encodePNG(draw(180)))
	var pngs [][]byte
	for _, s := range []int{16, 32, 48, 256} {
		pngs = append(pngs, encodePNG(draw(s)))
	}
	write("web/icon.ico", ico(pngs, []int{16, 32, 48, 256}))
}

func write(name string, b []byte) {
	if err := os.WriteFile(name, b, 0o644); err != nil {
		log.Fatal(err)
	}
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

// ico packs PNG images into a Windows .ico container (PNG entries are
// supported since Windows Vista).
func ico(pngs [][]byte, sizes []int) []byte {
	var buf bytes.Buffer
	le := binary.LittleEndian
	binary.Write(&buf, le, [3]uint16{0, 1, uint16(len(pngs))})
	offset := 6 + 16*len(pngs)
	for i, p := range pngs {
		s := sizes[i]
		if s >= 256 {
			s = 0 // 0 means 256 in ICO headers
		}
		buf.Write([]byte{byte(s), byte(s), 0, 0})
		binary.Write(&buf, le, uint16(1))
		binary.Write(&buf, le, uint16(32))
		binary.Write(&buf, le, uint32(len(p)))
		binary.Write(&buf, le, uint32(offset))
		offset += len(p)
	}
	for _, p := range pngs {
		buf.Write(p)
	}
	return buf.Bytes()
}

// draw renders the icon at size×size using 4×4 supersampling. Coordinates
// below are in a 0..1 unit square.
func draw(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	const ss = 4
	c1 := [3]float64{13, 127, 138} // #0d7f8a
	c2 := [3]float64{19, 163, 160} // #13a3a0
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var bg, fg float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/ss) / float64(size)
					y := (float64(py) + (float64(sy)+0.5)/ss) / float64(size)
					if inRoundRect(x, y, 0.03, 0.22) {
						bg++
						if inMark(x, y) {
							fg++
						}
					}
				}
			}
			n := float64(ss * ss)
			a := bg / n
			if a == 0 {
				continue
			}
			t := (float64(px) + float64(py)) / float64(2*size)
			r := c1[0] + (c2[0]-c1[0])*t
			g := c1[1] + (c2[1]-c1[1])*t
			b := c1[2] + (c2[2]-c1[2])*t
			w := fg / bg // share of white within covered area
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(r + (255-r)*w),
				G: uint8(g + (255-g)*w),
				B: uint8(b + (255-b)*w),
				A: uint8(a * 255),
			})
		}
	}
	return img
}

func inRoundRect(x, y, inset, radius float64) bool {
	lo, hi := inset, 1-inset
	if x < lo || x > hi || y < lo || y > hi {
		return false
	}
	cx := math.Max(lo+radius, math.Min(x, hi-radius))
	cy := math.Max(lo+radius, math.Min(y, hi-radius))
	return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= radius*radius
}

// inMark is the white ferry: hull, cabin, mast flag and two waves.
func inMark(x, y float64) bool {
	// Hull: trapezoid, wide on top.
	if y >= 0.50 && y <= 0.60 {
		k := (y - 0.50) / 0.10
		if x >= 0.24+0.07*k && x <= 0.76-0.07*k {
			return true
		}
	}
	// Cabin.
	if y >= 0.41 && y < 0.50 && x >= 0.36 && x <= 0.62 {
		return true
	}
	// Mast.
	if y >= 0.24 && y < 0.41 && x >= 0.475 && x <= 0.515 {
		return true
	}
	// Flag: triangle pointing right from the mast top.
	if x >= 0.515 && x <= 0.64 {
		k := (x - 0.515) / 0.125
		if y >= 0.24+0.045*k && y <= 0.33-0.045*k {
			return true
		}
	}
	// Waves.
	for _, w := range []struct{ y0, amp float64 }{{0.69, 0.025}, {0.79, 0.025}} {
		if x < 0.18 || x > 0.82 {
			continue
		}
		wy := w.y0 + w.amp*math.Sin((x-0.18)*2*math.Pi*2.2)
		if math.Abs(y-wy) <= 0.022 {
			return true
		}
	}
	return false
}
