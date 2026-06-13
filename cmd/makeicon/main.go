// Command makeicon generates the Diagnostic Studio application icon: a teal
// rounded tile with a white diagnostic pulse line. It writes a multi-size
// Windows .ico and the Wails appicon.png, using only the standard library so
// no image dependencies enter the build.
//
// Run once from the repo root:
//
//	go run ./cmd/makeicon
//
// Outputs:
//	build/windows/icon.ico
//	build/appicon.png
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Brand palette (matches frontend/src/style.css).
var (
	tealTop = color.NRGBA{0x12, 0x9b, 0x9b, 0xff} // slightly lighter top
	tealBot = color.NRGBA{0x0a, 0x6b, 0x6b, 0xff} // --teal-dark
	white   = color.NRGBA{0xff, 0xff, 0xff, 0xff}
)

func main() {
	// Master render at high resolution, then downscale to each icon size.
	const master = 256
	img := renderIcon(master)

	if err := writePNG(filepath.Join("build", "appicon.png"), img); err != nil {
		panic(err)
	}

	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	frames := make([]*image.NRGBA, 0, len(sizes))
	for _, s := range sizes {
		if s == master {
			frames = append(frames, img)
		} else {
			frames = append(frames, resize(img, s))
		}
	}
	if err := writeICO(filepath.Join("build", "windows", "icon.ico"), frames); err != nil {
		panic(err)
	}
}

// renderIcon draws the tile + pulse at the given square size with 4x
// supersampling for smooth edges.
func renderIcon(size int) *image.NRGBA {
	const ss = 4
	big := size * ss
	buf := image.NewNRGBA(image.Rect(0, 0, big, big))

	radius := float64(big) * 0.22
	for y := 0; y < big; y++ {
		for x := 0; x < big; x++ {
			fx, fy := float64(x), float64(y)
			if !insideRounded(fx, fy, float64(big), radius) {
				continue
			}
			// vertical teal gradient
			t := fy / float64(big)
			buf.SetNRGBA(x, y, lerp(tealTop, tealBot, t))
		}
	}

	drawPulse(buf, big)
	return resize(buf, size)
}

// drawPulse strokes a centered ECG-like line: flat, up-spike, down-spike,
// flat — the diagnostic motif.
func drawPulse(img *image.NRGBA, size int) {
	w := float64(size)
	midY := w * 0.52
	amp := w * 0.20
	stroke := w * 0.055

	// Control points across the width (fractions of width, y offset in amp).
	pts := []struct{ fx, fy float64 }{
		{0.16, 0}, {0.36, 0}, {0.44, -1.0}, {0.52, 1.0}, {0.60, 0}, {0.84, 0},
	}
	abs := func(v float64) float64 {
		if v < 0 {
			return -v
		}
		return v
	}
	for i := 0; i < len(pts)-1; i++ {
		x0, y0 := pts[i].fx*w, midY+pts[i].fy*amp
		x1, y1 := pts[i+1].fx*w, midY+pts[i+1].fy*amp
		steps := int(math.Hypot(x1-x0, y1-y0)) + 1
		for s := 0; s <= steps; s++ {
			f := float64(s) / float64(steps)
			cx := x0 + (x1-x0)*f
			cy := y0 + (y1-y0)*f
			r := stroke
			y0i := int(cy - r - 1)
			y1i := int(cy + r + 1)
			x0i := int(cx - r - 1)
			x1i := int(cx + r + 1)
			for yy := y0i; yy <= y1i; yy++ {
				for xx := x0i; xx <= x1i; xx++ {
					if xx < 0 || yy < 0 || xx >= size || yy >= size {
						continue
					}
					d := math.Hypot(float64(xx)-cx, float64(yy)-cy)
					if d <= r {
						img.SetNRGBA(xx, yy, white)
					} else if d <= r+1.2 && abs(d-r) > 0 {
						// soft edge
						a := 1 - (d-r)/1.2
						blendOver(img, xx, yy, white, a)
					}
				}
			}
		}
	}
}

func insideRounded(x, y, size, r float64) bool {
	if x >= r && x <= size-r {
		return y >= 0 && y <= size
	}
	if y >= r && y <= size-r {
		return x >= 0 && x <= size
	}
	cx := math.Max(r, math.Min(x, size-r))
	cy := math.Max(r, math.Min(y, size-r))
	return math.Hypot(x-cx, y-cy) <= r
}

func lerp(a, b color.NRGBA, t float64) color.NRGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.NRGBA{
		uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		0xff,
	}
}

func blendOver(img *image.NRGBA, x, y int, fg color.NRGBA, a float64) {
	if a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	bg := img.NRGBAAt(x, y)
	if bg.A == 0 {
		return // keep transparent corners transparent
	}
	img.SetNRGBA(x, y, color.NRGBA{
		uint8(float64(fg.R)*a + float64(bg.R)*(1-a)),
		uint8(float64(fg.G)*a + float64(bg.G)*(1-a)),
		uint8(float64(fg.B)*a + float64(bg.B)*(1-a)),
		0xff,
	})
}

// resize does a simple box-average downscale (src must be >= dst).
func resize(src *image.NRGBA, dst int) *image.NRGBA {
	sw := src.Bounds().Dx()
	out := image.NewNRGBA(image.Rect(0, 0, dst, dst))
	scale := float64(sw) / float64(dst)
	for y := 0; y < dst; y++ {
		for x := 0; x < dst; x++ {
			// Average over the source box. Colors are weighted by source
			// alpha so transparent corners don't darken the rounded edge.
			var r, g, b, aSum, wSum, n float64
			x0 := int(float64(x) * scale)
			y0 := int(float64(y) * scale)
			x1 := int(float64(x+1) * scale)
			y1 := int(float64(y+1) * scale)
			if x1 <= x0 {
				x1 = x0 + 1
			}
			if y1 <= y0 {
				y1 = y0 + 1
			}
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					c := src.NRGBAAt(xx, yy)
					af := float64(c.A) / 255
					r += float64(c.R) * af
					g += float64(c.G) * af
					b += float64(c.B) * af
					aSum += float64(c.A)
					wSum += af
					n++
				}
			}
			if n == 0 || aSum == 0 {
				continue
			}
			out.SetNRGBA(x, y, color.NRGBA{
				uint8(r / wSum),
				uint8(g / wSum),
				uint8(b / wSum),
				uint8(aSum / n),
			})
		}
	}
	return out
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// writeICO writes a Windows .ico containing PNG-encoded frames (Vista+ ICO
// supports embedded PNG; Wails/Windows handle this fine).
func writeICO(path string, frames []*image.NRGBA) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var dir bytes.Buffer
	var blobs [][]byte

	// ICONDIR header
	binary.Write(&dir, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&dir, binary.LittleEndian, uint16(1)) // type: icon
	binary.Write(&dir, binary.LittleEndian, uint16(len(frames)))

	offset := 6 + 16*len(frames)
	for _, fr := range frames {
		var buf bytes.Buffer
		if err := png.Encode(&buf, fr); err != nil {
			return err
		}
		data := buf.Bytes()
		blobs = append(blobs, data)

		sz := fr.Bounds().Dx()
		b := byte(sz)
		if sz >= 256 {
			b = 0 // 0 means 256 in ICO
		}
		dir.WriteByte(b)                                          // width
		dir.WriteByte(b)                                          // height
		dir.WriteByte(0)                                          // palette
		dir.WriteByte(0)                                          // reserved
		binary.Write(&dir, binary.LittleEndian, uint16(1))       // planes
		binary.Write(&dir, binary.LittleEndian, uint16(32))      // bpp
		binary.Write(&dir, binary.LittleEndian, uint32(len(data)))
		binary.Write(&dir, binary.LittleEndian, uint32(offset))
		offset += len(data)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(dir.Bytes()); err != nil {
		return err
	}
	for _, data := range blobs {
		if _, err := f.Write(data); err != nil {
			return err
		}
	}
	return nil
}
