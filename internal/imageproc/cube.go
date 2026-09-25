package imageproc

import (
	"fmt"
	"image"
)

func DetectCube(img image.Image, tol int) (*CubeGeom, func(int, int) bool, error) {
	b := img.Bounds()

	bgPoints := [][2]int{
		{b.Min.X, b.Min.Y}, {b.Max.X - 1, b.Min.Y},
		{b.Min.X, b.Max.Y - 1}, {b.Max.X - 1, b.Max.Y - 1},
	}

	var bgR, bgG, bgB, bgA uint32
	for _, p := range bgPoints {
		r, g, bl, a := img.At(p[0], p[1]).RGBA()
		if a == 0 {
			continue
		}
		bgR, bgG, bgB, bgA = r, g, bl, a
		break
	}

	cr8, cg8, cb8 := int(bgR>>8), int(bgG>>8), int(bgB>>8)
	alphaBG := bgA < 32768

	isBG := func(x, y int) bool {
		if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
			return true
		}
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
	scanCols := 5
	if maxx-minx+1 < scanCols {
		scanCols = maxx - minx + 1
	}
	for x := minx; x < minx+scanCols && x <= maxx; x++ {
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

	g := &CubeGeom{
		Htop: float64(ytop - miny),
		Edge: float64(ybot - ytop + 1),
		Hw:   float64(maxx-minx+1) / 2,
	}
	expect := 2*g.Htop + g.Edge
	actual := float64(maxy - miny + 1)
	if d := expect - actual; d < -3 || d > 3 {
		return nil, nil, fmt.Errorf("силуэт не похож на изо-куб (ожидал h=%.1f, реально %.1f)", expect, actual)
	}

	cx := float64(minx+maxx) / 2
	yD := float64(miny) + g.Htop
	yC := float64(miny) + 2*g.Htop
	g.Faces = map[string]Face{
		"top":   {O: [2]float64{float64(minx), yD}, U: [2]float64{g.Hw, g.Htop}, V: [2]float64{g.Hw, -g.Htop}},
		"left":  {O: [2]float64{float64(minx), yD}, U: [2]float64{g.Hw, g.Htop}, V: [2]float64{0, g.Edge}},
		"right": {O: [2]float64{cx, yC}, U: [2]float64{g.Hw, -g.Htop}, V: [2]float64{0, g.Edge}},
	}
	return g, isBG, nil
}

func SampleFace(img image.Image, isBG func(int, int) bool, f Face, n, ss int, inset float64) ([]Pixel, []bool) {
	b := img.Bounds()
	cells := make([]Pixel, n*n)
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
					sr += int(uint32(r) * 255 / uint32(a))
					sg += int(uint32(g) * 255 / uint32(a))
					sb += int(uint32(bl) * 255 / uint32(a))
					cnt++
				}
			}
			if cnt == 0 {
				continue
			}
			cells[v*n+u] = Pixel{uint8(sr / cnt), uint8(sg / cnt), uint8(sb / cnt)}
			opaque[v*n+u] = true
		}
	}
	return cells, opaque
}

func MeanLum(cells []Pixel, opaque []bool) float64 {
	var s float64
	var n int
	for i, ok := range opaque {
		if !ok {
			continue
		}
		s += 0.299*float64(cells[i].R) + 0.587*float64(cells[i].G) + 0.114*float64(cells[i].B)
		n++
	}
	if n == 0 {
		return 0
	}
	return s / float64(n)
}
