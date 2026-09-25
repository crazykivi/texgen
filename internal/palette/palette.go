package palette

import (
	"sort"
	"texgen/internal/imageproc"
)

type entry struct {
	c imageproc.Pixel
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
			v = e.c.R
		case 1:
			v = e.c.G
		default:
			v = e.c.B
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

func (b box) avg() imageproc.Pixel {
	var sr, sg, sb, w int
	for _, e := range b.es {
		sr += int(e.c.R) * e.n
		sg += int(e.c.G) * e.n
		sb += int(e.c.B) * e.n
		w += e.n
	}
	if w == 0 {
		w = 1
	}
	return imageproc.Pixel{uint8((sr + w/2) / w), uint8((sg + w/2) / w), uint8((sb + w/2) / w)}
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
			return es[i].c.R < es[j].c.R
		case 1:
			return es[i].c.G < es[j].c.G
		default:
			return es[i].c.B < es[j].c.B
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

func Quantize(cells []imageproc.Pixel, opaque []bool, k int) []imageproc.Pixel {
	counts := map[imageproc.Pixel]int{}
	for i, ok := range opaque {
		if ok {
			counts[cells[i]]++
		}
	}
	if len(counts) == 0 {
		return nil
	}

	entries := make([]entry, 0, len(counts))
	for c, n := range counts {
		entries = append(entries, entry{c, n})
	}
	if len(entries) <= k {
		pal := make([]imageproc.Pixel, len(entries))
		for i, e := range entries {
			pal[i] = e.c
		}
		return pal
	}

	boxes := []box{{es: entries}}
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
	pal := make([]imageproc.Pixel, 0, len(boxes))
	for _, b := range boxes {
		pal = append(pal, b.avg())
	}
	return pal
}
