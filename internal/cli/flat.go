package cli

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"texgen/internal/imageproc"
	"texgen/internal/svg"
)

func Process(path string, w, h, colors, px int, nearest bool, bright float64, outFlag string) {
	img, err := imageproc.LoadImg(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}

	svgStr := ProcessFlatInMemory(img, w, h, colors, px, nearest, bright)

	out := outFlag
	if out == "" {
		ext := filepath.Ext(path)
		out = strings.TrimSuffix(path, ext) + ".svg"
	}
	if err := os.WriteFile(out, []byte(svgStr), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "%s -> %s (grid %dx%d, bright %.2f)\n", path, out, w, h, bright)
}

func ProcessFlatInMemory(img image.Image, w, h, colors, px int, nearest bool, bright float64) string {
	cells, opaque := imageproc.Sample(img, w, h, nearest)
	imageproc.Adjust(cells, opaque, bright)
	return svg.Render(cells, opaque, w, h, colors, px)
}
