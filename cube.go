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

type cubeFace struct {
	O, U, V [2]float64
}

type cubeGeom struct {
	faces          map[string]cubeFace
	htop, edge, hw float64
}

func detectCube(img image.Image, tol int) (*cubeGeom, func(int, int) bool, error) {
	b := img.Bounds()
	cr, cg, cb, ca := img.At(b.Min.X, b.Min.Y).RGBA()
	cr8, cg8, cb8 := int(cr>>8), int(cg>>8), int(cb>>8)
	alphaBG := ca < 32768
	isBG := func(x, y int) bool {
		r, g, bl, a := img.At(x, y).RGBA()
		if a < 32768 {
			return true
		}
		if alphaBG {
			return false
		}
		dr := int(r>>8) - cr8
		dg := int(g>>8) - cg8
		db := int(bl>>8) - cb8
		return dr*dr+dg*dg+db*db <= tol*tol
	}

	minx, miny := b.Max.X, b.Max.Y
	maxx, maxy := b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !isBG(x, y) {
				if x < minx {
					minx = x
				}
				if x > maxx {
					maxx = x
				}
				if y < miny {
					miny = y
				}
				if y > maxy {
					maxy = y
				}
			}
		}
	}
	if maxx <= minx || maxy <= miny {
		return nil, nil, fmt.Errorf("не найден куб на картинке (проверь фон)")
	}

	ytop, ybot := b.Max.Y, b.Min.Y
	for x := minx; x <= minx+1 && x <= maxx; x++ {
		for y := miny; y <= maxy; y++ {
			if !isBG(x, y) {
				if y < ytop {
					ytop = y
				}
				if y > ybot {
					ybot = y
				}
			}
		}
	}
	g := &cubeGeom{
		htop: float64(ytop - miny),
		edge: float64(ybot - ytop + 1),
		hw:   float64(maxx-minx+1) / 2,
	}
	expect := 2*g.htop + g.edge
	actual := float64(maxy - miny + 1)
	if d := expect - actual; d < -3 || d > 3 {
		return nil, nil, fmt.Errorf("силуэт не похож на изо-куб (ожидал h=%.1f, реально %.1f)", expect, actual)
	}

	cx := float64(minx+maxx) / 2
	yD := float64(miny) + g.htop
	yC := float64(miny) + 2*g.htop
	g.faces = map[string]cubeFace{
		"top":   {O: [2]float64{float64(minx), yD}, U: [2]float64{g.hw, g.htop}, V: [2]float64{g.hw, -g.htop}},
		"left":  {O: [2]float64{float64(minx), yD}, U: [2]float64{g.hw, g.htop}, V: [2]float64{0, g.edge}},
		"right": {O: [2]float64{cx, yC}, U: [2]float64{g.hw, -g.htop}, V: [2]float64{0, g.edge}},
	}
	return g, isBG, nil
}

func sampleFace(img image.Image, isBG func(int, int) bool, f cubeFace, n, ss int, inset float64) ([]rgb, []bool) {
	b := img.Bounds()
	cells := make([]rgb, n*n)
	opaque := make([]bool, n*n)
	for v := 0; v < n; v++ {
		for u := 0; u < n; u++ {
			var sr, sg, sb, cnt int
			for sv := 0; sv < ss; sv++ {
				for su := 0; su < ss; su++ {
					fu := (float64(u) + (float64(su)+0.5)/float64(ss)) / float64(n)
					fv := (float64(v) + (float64(sv)+0.5)/float64(ss)) / float64(n)
					fu = inset + fu*(1-2*inset)
					fv = inset + fv*(1-2*inset)
					px := f.O[0] + fu*f.U[0] + fv*f.V[0]
					py := f.O[1] + fu*f.U[1] + fv*f.V[1]
					ix, iy := int(px+0.5), int(py+0.5)
					if ix < b.Min.X || ix >= b.Max.X || iy < b.Min.Y || iy >= b.Max.Y {
						continue
					}
					if isBG(ix, iy) {
						continue
					}
					r, g, bl, a := img.At(ix, iy).RGBA()
					if a == 0 {
						continue
					}
					sr += int(r) * 65535 / int(a)
					sg += int(g) * 65535 / int(a)
					sb += int(bl) * 65535 / int(a)
					cnt++
				}
			}
			if cnt == 0 {
				continue
			}
			cells[v*n+u] = rgb{uint8(sr / cnt >> 8), uint8(sg / cnt >> 8), uint8(sb / cnt >> 8)}
			opaque[v*n+u] = true
		}
	}
	return cells, opaque
}

func meanLum(cells []rgb, opaque []bool) float64 {
	var s float64
	var n int
	for i, ok := range opaque {
		if !ok {
			continue
		}
		s += 0.299*float64(cells[i].r) + 0.587*float64(cells[i].g) + 0.114*float64(cells[i].b)
		n++
	}
	if n == 0 {
		return 0
	}
	return s / float64(n)
}

func cubeSVG(cells []rgb, opaque []bool, n, colors, px int) string {
	counts := map[rgb]int{}
	for i, ok := range opaque {
		if ok {
			counts[cells[i]]++
		}
	}
	entries := make([]entry, 0, len(counts))
	for c, k := range counts {
		entries = append(entries, entry{c, k})
	}
	var pal []rgb
	if len(entries) <= colors {
		for _, e := range entries {
			pal = append(pal, e.c)
		}
	} else {
		pal = quantize(entries, colors)
	}
	idx := make([]int, n*n)
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
		n*px, n*px, n, n)
	for _, j := range order {
		p := pal[j]
		fmt.Fprintf(&sb, "    <!-- #%02x%02x%02x : %d px -->\n", p.r, p.g, p.b, palCnt[j])
		fmt.Fprintf(&sb, "    <g fill=\"#%02x%02x%02x\">\n", p.r, p.g, p.b)
		for y := 0; y < n; y++ {
			x := 0
			for x < n {
				if idx[y*n+x] != j {
					x++
					continue
				}
				x0 := x
				for x < n && idx[y*n+x] == j {
					x++
				}
				fmt.Fprintf(&sb, "        <rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"1\" />\n", x0, y, x-x0)
			}
		}
		sb.WriteString("    </g>\n")
	}
	sb.WriteString("</svg>\n")
	return sb.String()
}

func processCube(path, faceName string, n, colors, px, ss int, inset float64, unshade, mirror bool, bright float64, tol int, outFlag string) {
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
	g, isBG, err := detectCube(img, tol)
	if err != nil {
		fmt.Fprintln(os.Stderr, path, "error:", err)
		return
	}
	fc, ok := g.faces[faceName]
	if !ok {
		fmt.Fprintln(os.Stderr, "неизвестная грань:", faceName, "(нужно top|left|right)")
		return
	}

	cells, opaque := sampleFace(img, isBG, fc, n, ss, inset)

	factor := 1.0
	if unshade {
		var means []float64
		for _, name := range []string{"top", "left", "right"} {
			c, o := sampleFace(img, isBG, g.faces[name], 8, 2, inset)
			means = append(means, meanLum(c, o))
		}
		max := means[0]
		for _, m := range means {
			if m > max {
				max = m
			}
		}
		my := meanLum(cells, opaque)
		if my > 0 {
			factor = max / my
			if factor > 4 {
				factor = 4
			}
		}
		for i, ok := range opaque {
			if !ok {
				continue
			}
			c := cells[i]
			clamp := func(v int) uint8 {
				if v > 255 {
					return 255
				}
				return uint8(v)
			}
			cells[i] = rgb{
				clamp(int(float64(c.r)*factor + 0.5)),
				clamp(int(float64(c.g)*factor + 0.5)),
				clamp(int(float64(c.b)*factor + 0.5)),
			}
		}
	}

	if mirror {
		fl := make([]rgb, len(cells))
		fo := make([]bool, len(opaque))
		for v := 0; v < n; v++ {
			for u := 0; u < n; u++ {
				fl[v*n+(n-1-u)] = cells[v*n+u]
				fo[v*n+(n-1-u)] = opaque[v*n+u]
			}
		}
		cells, opaque = fl, fo
	}

	adjust(cells, opaque, bright)

	out := outFlag
	if out == "" {
		out = strings.TrimSuffix(path, extOf(path)) + "_" + faceName + ".svg"
	}
	svg := cubeSVG(cells, opaque, n, colors, px)
	if err := os.WriteFile(out, []byte(svg), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "%s: грань=%s сетка=%d цветов=%d unshade=x%.2f bright=%.2f -> %s\n", path, faceName, n, colors, factor, bright, out)
}

func processCubeInMemory(img image.Image, faceName string, n, colors, px, ss int, inset float64, unshade, mirror bool, bright float64, tol int) (string, error) {
	g, isBG, err := detectCube(img, tol)
	if err != nil {
		return "", err
	}
	fc, ok := g.faces[faceName]
	if !ok {
		return "", fmt.Errorf("неизвестная грань: %s", faceName)
	}
	cells, opaque := sampleFace(img, isBG, fc, n, ss, inset)

	if unshade {
		var means []float64
		for _, name := range []string{"top", "left", "right"} {
			c, o := sampleFace(img, isBG, g.faces[name], 8, 2, inset)
			means = append(means, meanLum(c, o))
		}
		max := means[0]
		for _, m := range means {
			if m > max {
				max = m
			}
		}
		my := meanLum(cells, opaque)
		if my > 0 {
			factor := max / my
			if factor > 4 {
				factor = 4
			}
			for i, ok := range opaque {
				if !ok {
					continue
				}
				c := cells[i]
				clamp := func(v int) uint8 {
					if v > 255 {
						return 255
					}
					return uint8(v)
				}
				cells[i] = rgb{
					clamp(int(float64(c.r)*factor + 0.5)),
					clamp(int(float64(c.g)*factor + 0.5)),
					clamp(int(float64(c.b)*factor + 0.5)),
				}
			}
		}
	}

	if mirror {
		fl := make([]rgb, len(cells))
		fo := make([]bool, len(opaque))
		for v := 0; v < n; v++ {
			for u := 0; u < n; u++ {
				fl[v*n+(n-1-u)] = cells[v*n+u]
				fo[v*n+(n-1-u)] = opaque[v*n+u]
			}
		}
		cells, opaque = fl, fo
	}

	adjust(cells, opaque, bright)
	return renderSVG(cells, opaque, n, n, colors, px), nil
}

func runCube(args []string) {
	fs := flag.NewFlagSet("cube", flag.ExitOnError)
	faceName := fs.String("face", "right", "грань: top|left|right")
	n := fs.Int("size", 16, "размер сетки текстуры")
	colors := fs.Int("colors", 4, "максимум цветов палитры")
	px := fs.Int("px", 4, "экранный размер пикселя в svg")
	ss := fs.Int("ss", 4, "сэмплов на тексель (качество усреднения)")
	inset := fs.Float64("inset", 0.04, "отступ от краёв грани, доля (срежает антиалиасинг)")
	unshade := fs.Bool("unshade", true, "компенсировать затенение грани (выкл: -unshade=false)")
	mirror := fs.Bool("mirror", false, "отзеркалить текстуру по горизонтали")
	bright := fs.Float64("bright", 1, "яркость снятой текстуры: <1 темнее, >1 светлее")
	tol := fs.Int("tol", 12, "допуск цвета фона для непрозрачных картинок (jpg)")
	out := fs.String("out", "", "файл вывода (по умолчанию имя_грань.svg)")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: texgen cube [flags] render.png ...")
		fs.PrintDefaults()
		os.Exit(1)
	}
	for _, p := range fs.Args() {
		processCube(p, *faceName, *n, *colors, *px, *ss, *inset, *unshade, *mirror, *bright, *tol, *out)
	}
}
