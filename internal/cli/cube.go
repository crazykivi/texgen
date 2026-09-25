package cli

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"texgen/internal/imageproc"
	"texgen/internal/svg"
)

func coreCubeProcess(img image.Image, faceName string, n, colors, px, ss int, inset float64, unshade, mirror bool, bright float64, tol int) (string, error) {
	g, isBG, err := imageproc.DetectCube(img, tol)
	if err != nil {
		return "", err
	}
	fc, ok := g.Faces[faceName]
	if !ok {
		return "", fmt.Errorf("неизвестная грань: %s", faceName)
	}

	cells, opaque := imageproc.SampleFace(img, isBG, fc, n, ss, inset)

	factor := 1.0
	if unshade {
		var means []float64
		for _, name := range []string{"top", "left", "right"} {
			c, o := imageproc.SampleFace(img, isBG, g.Faces[name], 8, 2, inset)
			means = append(means, imageproc.MeanLum(c, o))
		}
		max := means[0]
		for _, m := range means {
			if m > max {
				max = m
			}
		}
		my := imageproc.MeanLum(cells, opaque)
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
			cells[i] = imageproc.Pixel{
				clamp(int(float64(c.R)*factor + 0.5)),
				clamp(int(float64(c.G)*factor + 0.5)),
				clamp(int(float64(c.B)*factor + 0.5)),
			}
		}
	}

	if mirror {
		fl := make([]imageproc.Pixel, len(cells))
		fo := make([]bool, len(opaque))
		for v := 0; v < n; v++ {
			for u := 0; u < n; u++ {
				fl[v*n+(n-1-u)] = cells[v*n+u]
				fo[v*n+(n-1-u)] = opaque[v*n+u]
			}
		}
		cells, opaque = fl, fo
	}

	imageproc.Adjust(cells, opaque, bright)
	return svg.Render(cells, opaque, n, n, colors, px), nil
}

func processCube(path, faceName string, n, colors, px, ss int, inset float64, unshade, mirror bool, bright float64, tol int, outFlag string) {
	img, err := imageproc.LoadImg(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	svgStr, err := coreCubeProcess(img, faceName, n, colors, px, ss, inset, unshade, mirror, bright, tol)
	if err != nil {
		fmt.Fprintln(os.Stderr, path, "error:", err)
		return
	}

	out := outFlag
	if out == "" {
		ext := filepath.Ext(path)
		out = strings.TrimSuffix(path, ext) + "_" + faceName + ".svg"
	}
	if err := os.WriteFile(out, []byte(svgStr), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "%s: грань=%s сетка=%d -> %s\n", path, faceName, n, out)
}

func RunCube(args []string) {
	fs := flag.NewFlagSet("cube", flag.ExitOnError)
	faceName := fs.String("face", "right", "грань: top|left|right")
	n := fs.Int("size", 16, "размер сетки текстуры")
	colors := fs.Int("colors", 4, "максимум цветов палитры")
	px := fs.Int("px", 4, "экранный размер пикселя в svg")
	ss := fs.Int("ss", 4, "сэмплов на тексель (качество усреднения)")
	inset := fs.Float64("inset", 0.04, "отступ от краев грани, доля (срезает антиалиасинг)")
	unshade := fs.Bool("unshade", true, "компенсировать затемнение грани (выкл: -unshade=false)")
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

func ProcessCubeInMemory(img image.Image, faceName string, n, colors, px, ss int, inset float64, unshade, mirror bool, bright float64, tol int) (string, error) {
	return coreCubeProcess(img, faceName, n, colors, px, ss, inset, unshade, mirror, bright, tol)
}
