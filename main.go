package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sort"
	"strings"
)

type rgb struct{ r, g, b uint8 }

type entry struct {
	c rgb
	n int
}

type box struct{ es []entry }

func (b box) weight() int {
	w := 0
	for _, e := range b.es {
		w += e.n
	}
	return w
}

func (b box) minMax(ch int) (uint8, uint8) {
	mn, mx := uint8(255), uint8(0)
	for _, e := range b.es {
		var v uint8
		switch ch {
		case 0:
			v = e.c.r
		case 1:
			v = e.c.g
		default:
			v = e.c.b
		}
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	return mn, mx
}

func (b box) longest() (int, uint8) {
	best, bestR := 0, uint8(0)
	for ch := 0; ch < 3; ch++ {
		mn, mx := b.minMax(ch)
		if mx-mn > bestR {
			best, bestR = ch, mx-mn
		}
	}
	return best, bestR
}

func (b box) avg() rgb {
	var sr, sg, sb, w int
	for _, e := range b.es {
		sr += int(e.c.r) * e.n
		sg += int(e.c.g) * e.n
		sb += int(e.c.b) * e.n
		w += e.n
	}
	if w == 0 {
		w = 1
	}
	return rgb{uint8((sr + w/2) / w), uint8((sg + w/2) / w), uint8((sb + w/2) / w)}
}

func split(b box) (box, box, bool) {
	ch, rng := b.longest()
	if rng == 0 || len(b.es) < 2 {
		return box{}, box{}, false
	}
	es := make([]entry, len(b.es))
	copy(es, b.es)
	sort.Slice(es, func(i, j int) bool {
		switch ch {
		case 0:
			return es[i].c.r < es[j].c.r
		case 1:
			return es[i].c.g < es[j].c.g
		default:
			return es[i].c.b < es[j].c.b
		}
	})
	half, acc, i := b.weight()/2, 0, 0
	for i < len(es)-1 {
		acc += es[i].n
		i++
		if acc >= half {
			break
		}
	}
	return box{es: es[:i]}, box{es: es[i:]}, true
}

// quantize — median cut по взвешенным уникальным цветам сетки
func quantize(es []entry, k int) []rgb {
	boxes := []box{{es: es}}
	for len(boxes) < k {
		best, bestScore := -1, -1
		for i, b := range boxes {
			_, rng := b.longest()
			if rng == 0 || len(b.es) < 2 {
				continue
			}
			if score := int(rng) * b.weight(); score > bestScore {
				bestScore, best = score, i
			}
		}
		if best < 0 {
			break
		}
		b1, b2, _ := split(boxes[best])
		boxes = append(boxes[:best], boxes[best+1:]...)
		boxes = append(boxes, b1, b2)
	}
	pal := make([]rgb, 0, len(boxes))
	for _, b := range boxes {
		pal = append(pal, b.avg())
	}
	return pal
}

// sample — даунскейл до w*h ячеек: усреднение по площади либо nearest
func sample(img image.Image, w, h int, nearest bool) ([]rgb, []bool) {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	cells := make([]rgb, w*h)
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
				cells[y*w+x] = rgb{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}
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
					sr += int(r) * 65535 / int(a)
					sg += int(g) * 65535 / int(a)
					sb += int(bl) * 65535 / int(a)
					sa += int(a)
					n++
				}
			}
			if n == 0 || sa/n < 32768 {
				continue // ячейка прозрачная -> rect не пишется
			}
			cells[y*w+x] = rgb{uint8(sr / n >> 8), uint8(sg / n >> 8), uint8(sb / n >> 8)}
			opaque[y*w+x] = true
		}
	}
	return cells, opaque
}

// adjust — яркость: <1 темнее (мультипликативно), >1 светлее (без клипования светов)
func adjust(cells []rgb, opaque []bool, bright float64) {
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
			r = int(float64(c.r) * bright)
			g = int(float64(c.g) * bright)
			b = int(float64(c.b) * bright)
		} else {
			r = 255 - int(float64(255-int(c.r))/bright)
			g = 255 - int(float64(255-int(c.g))/bright)
			b = 255 - int(float64(255-int(c.b))/bright)
		}
		cells[i] = rgb{uint8(r), uint8(g), uint8(b)}
	}
}

func process(path string, w, h, colors, px int, nearest bool, bright float64, outFlag string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}

	cells, opaque := sample(img, w, h, nearest)
	adjust(cells, opaque, bright)

	counts := map[rgb]int{}
	for i, ok := range opaque {
		if ok {
			counts[cells[i]]++
		}
	}
	entries := make([]entry, 0, len(counts))
	for c, n := range counts {
		entries = append(entries, entry{c, n})
	}

	var pal []rgb
	if len(entries) <= colors {
		for _, e := range entries { // цветов мало (пиксель-арт) — берётся, как есть
			pal = append(pal, e.c)
		}
	} else {
		pal = quantize(entries, colors) // фото — медиан-кат
	}

	idx := make([]int, w*h)
	palCnt := make([]int, len(pal))
	for i := range idx {
		idx[i] = -1
	}
	for i, ok := range opaque {
		if !ok {
			continue
		}
		best, bd := 0, 1<<30
		for j, p := range pal {
			dr := int(cells[i].r) - int(p.r)
			dg := int(cells[i].g) - int(p.g)
			db := int(cells[i].b) - int(p.b)
			if d := dr*dr + dg*dg + db*db; d < bd {
				bd, best = d, j
			}
		}
		idx[i] = best
		palCnt[best]++
	}

	order := make([]int, 0, len(pal))
	for j := range pal {
		if palCnt[j] > 0 {
			order = append(order, j)
		}
	}
	sort.Slice(order, func(a, b int) bool { return palCnt[order[a]] > palCnt[order[b]] })

	var sb strings.Builder
	fmt.Fprintf(&sb, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\"\n    shape-rendering=\"crispEdges\">\n",
		w*px, h*px, w, h)
	for _, j := range order {
		p := pal[j]
		fmt.Fprintf(&sb, "    <!-- #%02x%02x%02x : %d px -->\n", p.r, p.g, p.b, palCnt[j])
		fmt.Fprintf(&sb, "    <g fill=\"#%02x%02x%02x\">\n", p.r, p.g, p.b)
		for y := 0; y < h; y++ {
			x := 0
			for x < w {
				if idx[y*w+x] != j {
					x++
					continue
				}
				x0 := x
				for x < w && idx[y*w+x] == j {
					x++
				}
				fmt.Fprintf(&sb, "        <rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"1\" />\n", x0, y, x-x0)
			}
		}
		sb.WriteString("    </g>\n")
	}
	sb.WriteString("</svg>\n")

	out := outFlag
	if out == "" {
		out = strings.TrimSuffix(path, extOf(path)) + ".svg"
	}
	if err := os.WriteFile(out, []byte(sb.String()), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "%s -> %s (grid %dx%d, colors %d, bright %.2f)\n", path, out, w, h, len(order), bright)
}

func extOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '.' {
			return p[i:]
		}
		if p[i] == '/' || p[i] == '\\' {
			break
		}
	}
	return ""
}

func main() {
	w := flag.Int("w", 16, "ширина сетки текстуры")
	h := flag.Int("h", 16, "высота сетки текстуры")
	colors := flag.Int("colors", 4, "максимум цветов палитры (квантование фото)")
	px := flag.Int("px", 4, "размер одного пикселя на экране (width = w*px)")
	nearest := flag.Bool("nearest", false, "nearest-neighbor вместо усреднения (для пиксель-арта)")
	bright := flag.Float64("bright", 1, "яркость: <1 темнее (0.6 = -40%), >1 светлее")
	out := flag.String("out", "", "файл вывода (по умолчанию имя_входа.svg)")
	flag.Parse()

	args := flag.Args()
	if len(args) >= 1 && args[0] == "cube" {
		runCube(args[1:])
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: texgen [flags] image.png ... | texgen cube [flags] render.png")
		flag.PrintDefaults()
		os.Exit(1)
	}
	for _, p := range args {
		process(p, *w, *h, *colors, *px, *nearest, *bright, *out)
	}
}
