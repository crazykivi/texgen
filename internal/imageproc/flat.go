package imageproc

import "image"

func Sample(img image.Image, w, h int, nearest bool) ([]Pixel, []bool) {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	cells := make([]Pixel, w*h)
	opaque := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			x0 := b.Min.X + x*sw/w
			x1 := b.Min.X + (x+1)*sw/w
			y0 := b.Min.Y + y*sh/h
			y1 := b.Min.Y + (y+1)*sh/h
			if x1 <= x0 {
				x1 = x0 + 1
			}
			if y1 <= y0 {
				y1 = y0 + 1
			}

			if nearest {
				cx, cy := (x0+x1)/2, (y0+y1)/2
				if cx >= b.Max.X {
					cx = b.Max.X - 1
				}
				if cy >= b.Max.Y {
					cy = b.Max.Y - 1
				}
				r, g, bl, a := img.At(cx, cy).RGBA()
				if a == 0 {
					continue
				}
				cells[y*w+x] = Pixel{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}
				opaque[y*w+x] = true
				continue
			}

			var sr, sg, sb, sa, n int
			for py := y0; py < y1 && py < b.Max.Y; py++ {
				for px := x0; px < x1 && px < b.Max.X; px++ {
					r, g, bl, a := img.At(px, py).RGBA()
					if a == 0 {
						continue
					}
					// FIX: Безопасное умножение без overflow на 32-битных системах
					sr += int(uint32(r) * 255 / uint32(a))
					sg += int(uint32(g) * 255 / uint32(a))
					sb += int(uint32(bl) * 255 / uint32(a))
					sa += int(a)
					n++
				}
			}
			if n == 0 || sa/n < 32768 {
				continue
			}
			cells[y*w+x] = Pixel{uint8(sr / n), uint8(sg / n), uint8(sb / n)}
			opaque[y*w+x] = true
		}
	}
	return cells, opaque
}

func Adjust(cells []Pixel, opaque []bool, bright float64) {
	if bright == 1 {
		return
	}
	for i, ok := range opaque {
		if !ok {
			continue
		}
		c := cells[i]
		var r, g, b int
		if bright < 1 {
			r = int(float64(c.R) * bright)
			g = int(float64(c.G) * bright)
			b = int(float64(c.B) * bright)
		} else {
			r = 255 - int(float64(255-int(c.R))/bright)
			g = 255 - int(float64(255-int(c.G))/bright)
			b = 255 - int(float64(255-int(c.B))/bright)
		}
		cells[i] = Pixel{uint8(r), uint8(g), uint8(b)}
	}
}
