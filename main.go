package main

import (
	"flag"
	"texgen/internal/cli"
	"texgen/internal/gui"
)

func main() {
	w := flag.Int("w", 16, "Ширина сетки текстуры")
	h := flag.Int("h", 16, "высота сетки текстуры")
	colors := flag.Int("colors", 4, "максимум цветов палитры")
	px := flag.Int("px", 4, "размер пикселя на экране (width = w*px)")
	nearest := flag.Bool("nearest", false, "nearest-neighbor вместо усреднения")
	bright := flag.Float64("bright", 1, "яркость: <1 темнее, >1 светлее")
	out := flag.String("out", "", "файл вывода (по умолчанию имя_входа.svg)")
	flag.Parse()

	args := flag.Args()

	if len(args) >= 1 && args[0] == "cube" {
		cli.RunCube(args[1:])
		return
	}
	if len(args) >= 1 && args[0] == "gui" {
		gui.RunGUI()
		return
	}

	if len(args) < 1 {
		gui.RunGUI()
		return
	}

	for _, p := range args {
		cli.Process(p, *w, *h, *colors, *px, *nearest, *bright, *out)
	}
}
