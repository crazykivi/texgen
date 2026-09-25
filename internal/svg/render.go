package svg

import (
	"sort"
	"strconv"
	"strings"
	"texgen/internal/imageproc"
	"texgen/internal/palette"
)

func Render(cells []imageproc.Pixel, opaque []bool, w, h, colors, px int) string {
	pal := palette.Quantize(cells, opaque, colors)
	if len(pal) == 0 {
		return emptySVG(w, h, px)
	}

	idx := make([]int, len(cells))
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
			dr := int(cells[i].R) - int(p.R)
			dg := int(cells[i].G) - int(p.G)
			db := int(cells[i].B) - int(p.B)
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
	sb.Grow(w * h * 15)

	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="`)
	sb.WriteString(strconv.Itoa(w * px))
	sb.WriteString(`" height="`)
	sb.WriteString(strconv.Itoa(h * px))
	sb.WriteString(`" viewBox="0 0 `)
	sb.WriteString(strconv.Itoa(w))
	sb.WriteString(` `)
	sb.WriteString(strconv.Itoa(h))
	sb.WriteString(`"` + "\n")
	sb.WriteString(`    shape-rendering="crispEdges">` + "\n")

	for _, j := range order {
		p := pal[j]
		sb.WriteString("    <!-- #")
		sb.WriteString(hex(p.R))
		sb.WriteString(hex(p.G))
		sb.WriteString(hex(p.B))
		sb.WriteString(" : ")
		sb.WriteString(strconv.Itoa(palCnt[j]))
		sb.WriteString(" px -->\n")

		sb.WriteString("    <g fill=\"#")
		sb.WriteString(hex(p.R))
		sb.WriteString(hex(p.G))
		sb.WriteString(hex(p.B))
		sb.WriteString("\">\n")

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
				sb.WriteString(`        <rect x="`)
				sb.WriteString(strconv.Itoa(x0))
				sb.WriteString(`" y="`)
				sb.WriteString(strconv.Itoa(y))
				sb.WriteString(`" width="`)
				sb.WriteString(strconv.Itoa(x - x0))
				sb.WriteString(`" height="1" />` + "\n")
			}
		}
		sb.WriteString("    </g>\n")
	}
	sb.WriteString("</svg>\n")
	return sb.String()
}

func hex(v uint8) string {
	const hextable = "0123456789abcdef"
	return string([]byte{hextable[v>>4], hextable[v&0x0f]})
}

func emptySVG(w, h, px int) string {
	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="`)
	sb.WriteString(strconv.Itoa(w * px))
	sb.WriteString(`" height="`)
	sb.WriteString(strconv.Itoa(h * px))
	sb.WriteString(`" viewBox="0 0 `)
	sb.WriteString(strconv.Itoa(w))
	sb.WriteString(` `)
	sb.WriteString(strconv.Itoa(h))
	sb.WriteString(`" shape-rendering="crispEdges"></svg>`)
	return sb.String()
}
