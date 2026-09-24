package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type cyrillicTheme struct{}

var cyrillicFont fyne.Resource

func initCyrillicFont() {
	candidates := []string{
		`C:\Windows\Fonts\segoeui.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
		`C:\Windows\Fonts\calibri.ttf`,
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			cyrillicFont = fyne.NewStaticResource(filepath.Base(p), data)
			return
		}
	}
	cyrillicFont = nil
}

func (cyrillicTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}
func (cyrillicTheme) Font(_ fyne.TextStyle) fyne.Resource {
	if cyrillicFont != nil {
		return cyrillicFont
	}
	return theme.DefaultTheme().Font(fyne.TextStyle{})
}
func (cyrillicTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}
func (cyrillicTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}

type preset struct {
	Name    string  `json:"name"`
	Mode    string  `json:"mode"`
	W       int     `json:"w"`
	H       int     `json:"h"`
	Colors  int     `json:"colors"`
	Px      int     `json:"px"`
	Nearest bool    `json:"nearest"`
	Bright  float64 `json:"bright"`
	Face    string  `json:"face"`
	Size    int     `json:"size"`
	SS      int     `json:"ss"`
	Inset   float64 `json:"inset"`
	Unshade bool    `json:"unshade"`
	Mirror  bool    `json:"mirror"`
	Tol     int     `json:"tol"`
}

func presetsPath() string {
	cwd, _ := os.Getwd()
	if cwd != "" {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return filepath.Join(cwd, "presets.json")
		}
	}
	exe, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exe), "presets.json")
	}
	return filepath.Join(cwd, "presets.json")
}

func loadPresets() []preset {
	b, err := os.ReadFile(presetsPath())
	if err != nil {
		return nil
	}
	var ps []preset
	_ = json.Unmarshal(b, &ps)
	out := make([]preset, 0, len(ps))
	for _, p := range ps {
		if strings.TrimSpace(p.Name) == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func savePresets(ps []preset) {
	b, _ := json.MarshalIndent(ps, "", "  ")
	_ = os.WriteFile(presetsPath(), b, 0644)
}

func readInt(e *widget.Entry, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(e.Text))
	if err != nil {
		return def
	}
	return n
}
func readFloat(e *widget.Entry, def float64) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(e.Text), 64)
	if err != nil {
		return def
	}
	return n
}

var previewSeq int

func newPreview() *canvas.Image {
	img := &canvas.Image{}
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(256, 256))
	return img
}

func setPreview(img *canvas.Image, svg string) {
	previewSeq++
	res := fyne.NewStaticResource(fmt.Sprintf("preview_%d.svg", previewSeq), []byte(svg))
	img.Resource = res
	img.Refresh()
}

func uriPath(u fyne.URI) string {
	s := u.String()
	s = strings.TrimPrefix(s, "file://")
	if dec, err := url.PathUnescape(s); err == nil {
		s = dec
	}
	if len(s) >= 3 && (s[0] == '/' || s[0] == '\\') && s[2] == ':' {
		s = s[1:]
	}
	return filepath.FromSlash(s)
}

func ensureExt(path string) string {
	if path == "" {
		return path
	}
	if !strings.EqualFold(filepath.Ext(path), ".svg") {
		return path + ".svg"
	}
	return path
}

func saveToFile(w fyne.Window, svg string, onDone func(string)) {
	dlg := dialog.NewFileSave(func(u fyne.URIWriteCloser, err error) {
		if err != nil || u == nil {
			return
		}
		_ = u.Close()
		path := ensureExt(uriPath(u.URI()))
		if err := os.WriteFile(path, []byte(svg), 0644); err != nil {
			dialog.ShowError(err, w)
			return
		}
		if onDone != nil {
			onDone(path)
		}
	}, w)
	dlg.SetFileName("texture.svg")
	dlg.Show()
}

func newSpinBox(initial, step, minVal, maxVal int) (*widget.Entry, fyne.CanvasObject) {
	e := widget.NewEntry()
	e.SetText(fmt.Sprint(initial))
	update := func(delta int) {
		v := readInt(e, initial) + delta
		if v < minVal {
			v = minVal
		}
		if maxVal > 0 && v > maxVal {
			v = maxVal
		}
		e.SetText(fmt.Sprint(v))
	}
	up := widget.NewButton("▲", func() { update(step) })
	down := widget.NewButton("▼", func() { update(-step) })
	up.Importance = widget.LowImportance
	down.Importance = widget.LowImportance
	return e, container.NewBorder(nil, nil, nil, container.NewVBox(up, down), e)
}

func newFloatSpinBox(initial, step, minVal, maxVal float64) (*widget.Entry, fyne.CanvasObject) {
	e := widget.NewEntry()
	e.SetText(fmt.Sprintf("%.2f", initial))
	update := func(delta float64) {
		v := readFloat(e, initial) + delta
		if v < minVal {
			v = minVal
		}
		if maxVal > 0 && v > maxVal {
			v = maxVal
		}
		e.SetText(fmt.Sprintf("%.2f", v))
	}
	up := widget.NewButton("▲", func() { update(step) })
	down := widget.NewButton("▼", func() { update(-step) })
	up.Importance = widget.LowImportance
	down.Importance = widget.LowImportance
	return e, container.NewBorder(nil, nil, nil, container.NewVBox(up, down), e)
}

type debouncer struct {
	mu    sync.Mutex
	timer *time.Timer
}

func (d *debouncer) call(delay time.Duration, f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(delay, f)
}

func runGUI() {
	initCyrillicFont()

	a := app.NewWithID("ru.texgen.svggen")
	a.Settings().SetTheme(cyrillicTheme{})
	w := a.NewWindow("texgen — генератор SVG-текстур")
	w.Resize(fyne.NewSize(1100, 700))

	imgFilter := storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".gif"})

	flatDebounce := &debouncer{}
	cubeDebounce := &debouncer{}
	const debounceDelay = 300 * time.Millisecond

	flatActivity := widget.NewActivity()
	flatActivity.Hide()
	cubeActivity := widget.NewActivity()
	cubeActivity.Hide()

	var selectTab func(mode string)

	fWEntry, fWBox := newSpinBox(16, 1, 1, 256)
	fHEntry, fHBox := newSpinBox(16, 1, 1, 256)
	fColorsEntry, fColorsBox := newSpinBox(4, 1, 1, 256)
	fPxEntry, fPxBox := newSpinBox(4, 1, 1, 64)
	fBrightEntry, fBrightBox := newFloatSpinBox(1.0, 0.05, 0.0, 5.0)

	fNearest := widget.NewCheck("nearest-neighbor (пиксель-арт)", nil)
	fSrc := widget.NewEntry()
	fSrc.SetPlaceHolder("путь к файлу... или перетащи картинку в окно")
	fPreview := newPreview()
	fLog := widget.NewMultiLineEntry()
	fLog.SetText("готов: выбери файл или перетащи его в окно")
	fLog.Wrapping = fyne.TextWrapWord

	refreshFlat := func() {
		src := fSrc.Text
		if src == "" {
			return
		}
		fyne.Do(func() {
			fLog.SetText("обработка...")
			flatActivity.Show()
			flatActivity.Start()
		})
		img, err := loadImg(src)
		if err != nil {
			fyne.Do(func() {
				flatActivity.Stop()
				flatActivity.Hide()
				fLog.SetText("файл не читается: " + err.Error())
			})
			return
		}
		svg := processInMemory(img, readInt(fWEntry, 16), readInt(fHEntry, 16),
			readInt(fColorsEntry, 4), readInt(fPxEntry, 4),
			fNearest.Checked, readFloat(fBrightEntry, 1))
		fyne.Do(func() {
			setPreview(fPreview, svg)
			fLog.SetText(fmt.Sprintf("готово: %dx%d, цветов=%d, bright=%.2f",
				readInt(fWEntry, 16), readInt(fHEntry, 16), readInt(fColorsEntry, 4), readFloat(fBrightEntry, 1)))
			flatActivity.Stop()
			flatActivity.Hide()
		})
	}
	scheduleFlat := func() { flatDebounce.call(debounceDelay, refreshFlat) }

	fSrcBtn := widget.NewButton("Обзор", func() {
		dlg := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err == nil && r != nil {
				p := uriPath(r.URI())
				_ = r.Close()
				fSrc.SetText(p)
				go refreshFlat()
			}
		}, w)
		dlg.SetFilter(imgFilter)
		dlg.Show()
	})
	fSaveBtn := widget.NewButton("Сохранить SVG", func() {
		src := fSrc.Text
		if src == "" {
			fLog.SetText("выбери файл")
			return
		}
		img, err := loadImg(src)
		if err != nil {
			fLog.SetText("ошибка: " + err.Error())
			return
		}
		svg := processInMemory(img, readInt(fWEntry, 16), readInt(fHEntry, 16),
			readInt(fColorsEntry, 4), readInt(fPxEntry, 4),
			fNearest.Checked, readFloat(fBrightEntry, 1))
		saveToFile(w, svg, func(path string) { fLog.SetText("сохранено: " + path) })
	})

	fWEntry.OnChanged = func(string) { scheduleFlat() }
	fHEntry.OnChanged = func(string) { scheduleFlat() }
	fColorsEntry.OnChanged = func(string) { scheduleFlat() }
	fPxEntry.OnChanged = func(string) { scheduleFlat() }
	fBrightEntry.OnChanged = func(string) { scheduleFlat() }
	fNearest.OnChanged = func(bool) { scheduleFlat() }
	fSrc.OnChanged = func(string) { scheduleFlat() }

	flatForm := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("ширина (w)", fWBox),
			widget.NewFormItem("высота (h)", fHBox),
			widget.NewFormItem("цветов", fColorsBox),
			widget.NewFormItem("px (размер)", fPxBox),
			widget.NewFormItem("яркость", fBrightBox),
		),
		fNearest,
		container.NewBorder(nil, nil, nil, fSrcBtn, fSrc),
		container.NewHBox(fSaveBtn, flatActivity),
		fLog,
	)
	flatView := container.NewHSplit(flatForm, container.NewCenter(fPreview))

	cFace := widget.NewSelect([]string{"top", "left", "right"}, nil)
	cFace.SetSelected("right")
	cSizeEntry, cSizeBox := newSpinBox(16, 1, 1, 256)
	cColorsEntry, cColorsBox := newSpinBox(4, 1, 1, 256)
	cPxEntry, cPxBox := newSpinBox(4, 1, 1, 64)
	cSSEntry, cSSBox := newSpinBox(4, 1, 1, 16)
	cInsetEntry, cInsetBox := newFloatSpinBox(0.04, 0.01, 0.0, 0.5)
	cTolEntry, cTolBox := newSpinBox(12, 1, 0, 255)
	cBrightEntry, cBrightBox := newFloatSpinBox(1.0, 0.05, 0.0, 5.0)

	cUnshade := widget.NewCheck("компенсировать затенение", nil)
	cUnshade.Checked = true
	cMirror := widget.NewCheck("зеркалить по горизонтали", nil)
	cSrc := widget.NewEntry()
	cSrc.SetPlaceHolder("путь к рендеру куба... или перетащи картинку в окно")
	cPreview := newPreview()
	cLog := widget.NewMultiLineEntry()
	cLog.SetText("готов: выбери рендер или перетащи его в окно")
	cLog.Wrapping = fyne.TextWrapWord

	refreshCube := func() {
		src := cSrc.Text
		if src == "" {
			return
		}
		fyne.Do(func() {
			cLog.SetText("обработка...")
			cubeActivity.Show()
			cubeActivity.Start()
		})
		img, err := loadImg(src)
		if err != nil {
			fyne.Do(func() {
				cubeActivity.Stop()
				cubeActivity.Hide()
				cLog.SetText("файл не читается: " + err.Error())
			})
			return
		}
		svg, err := processCubeInMemory(img, cFace.Selected,
			readInt(cSizeEntry, 16), readInt(cColorsEntry, 4), readInt(cPxEntry, 4), readInt(cSSEntry, 4),
			readFloat(cInsetEntry, 0.04), cUnshade.Checked, cMirror.Checked,
			readFloat(cBrightEntry, 1), readInt(cTolEntry, 12))
		if err != nil {
			fyne.Do(func() {
				cubeActivity.Stop()
				cubeActivity.Hide()
				cLog.SetText("ошибка: " + err.Error())
			})
			return
		}
		fyne.Do(func() {
			setPreview(cPreview, svg)
			cLog.SetText(fmt.Sprintf("готово: грань=%s, цветов=%d, bright=%.2f",
				cFace.Selected, readInt(cColorsEntry, 4), readFloat(cBrightEntry, 1)))
			cubeActivity.Stop()
			cubeActivity.Hide()
		})
	}
	scheduleCube := func() { cubeDebounce.call(debounceDelay, refreshCube) }

	cSrcBtn := widget.NewButton("Обзор", func() {
		dlg := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err == nil && r != nil {
				p := uriPath(r.URI())
				_ = r.Close()
				cSrc.SetText(p)
				go refreshCube()
			}
		}, w)
		dlg.SetFilter(imgFilter)
		dlg.Show()
	})
	cSaveBtn := widget.NewButton("Сохранить SVG", func() {
		src := cSrc.Text
		if src == "" {
			cLog.SetText("выбери файл")
			return
		}
		img, err := loadImg(src)
		if err != nil {
			cLog.SetText("ошибка: " + err.Error())
			return
		}
		svg, err := processCubeInMemory(img, cFace.Selected,
			readInt(cSizeEntry, 16), readInt(cColorsEntry, 4), readInt(cPxEntry, 4), readInt(cSSEntry, 4),
			readFloat(cInsetEntry, 0.04), cUnshade.Checked, cMirror.Checked,
			readFloat(cBrightEntry, 1), readInt(cTolEntry, 12))
		if err != nil {
			cLog.SetText("ошибка: " + err.Error())
			return
		}
		saveToFile(w, svg, func(path string) { cLog.SetText("сохранено: " + path) })
	})

	cFace.OnChanged = func(string) { scheduleCube() }
	cSizeEntry.OnChanged = func(string) { scheduleCube() }
	cColorsEntry.OnChanged = func(string) { scheduleCube() }
	cPxEntry.OnChanged = func(string) { scheduleCube() }
	cSSEntry.OnChanged = func(string) { scheduleCube() }
	cInsetEntry.OnChanged = func(string) { scheduleCube() }
	cBrightEntry.OnChanged = func(string) { scheduleCube() }
	cTolEntry.OnChanged = func(string) { scheduleCube() }
	cUnshade.OnChanged = func(bool) { scheduleCube() }
	cMirror.OnChanged = func(bool) { scheduleCube() }
	cSrc.OnChanged = func(string) { scheduleCube() }

	cubeForm := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("грань", cFace),
			widget.NewFormItem("размер сетки", cSizeBox),
			widget.NewFormItem("цветов", cColorsBox),
			widget.NewFormItem("px", cPxBox),
			widget.NewFormItem("сэмплов (ss)", cSSBox),
			widget.NewFormItem("inset", cInsetBox),
			widget.NewFormItem("tol (фон)", cTolBox),
			widget.NewFormItem("яркость", cBrightBox),
		),
		cUnshade, cMirror,
		container.NewBorder(nil, nil, nil, cSrcBtn, cSrc),
		container.NewHBox(cSaveBtn, cubeActivity),
		cLog,
	)
	cubeView := container.NewHSplit(cubeForm, container.NewCenter(cPreview))

	selectedPresetID := -1
	presetList := widget.NewList(
		func() int { return len(loadPresets()) },
		func() fyne.CanvasObject { return widget.NewLabel("preset template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			ps := loadPresets()
			if i >= len(ps) {
				return
			}
			o.(*widget.Label).SetText(fmt.Sprintf("%s   [%s]", ps[i].Name, ps[i].Mode))
		},
	)

	applyPreset := func(id int) {
		ps := loadPresets()
		if id < 0 || id >= len(ps) {
			return
		}
		p := ps[id]
		if p.Mode == "flat" {
			fWEntry.SetText(fmt.Sprint(p.W))
			fHEntry.SetText(fmt.Sprint(p.H))
			fColorsEntry.SetText(fmt.Sprint(p.Colors))
			fPxEntry.SetText(fmt.Sprint(p.Px))
			fBrightEntry.SetText(fmt.Sprintf("%.2f", p.Bright))
			fNearest.Checked = p.Nearest
			if selectTab != nil {
				selectTab("flat")
			}
			go refreshFlat()
		} else {
			cFace.SetSelected(p.Face)
			cSizeEntry.SetText(fmt.Sprint(p.Size))
			cColorsEntry.SetText(fmt.Sprint(p.Colors))
			cPxEntry.SetText(fmt.Sprint(p.Px))
			cSSEntry.SetText(fmt.Sprint(p.SS))
			cInsetEntry.SetText(fmt.Sprintf("%.2f", p.Inset))
			cBrightEntry.SetText(fmt.Sprintf("%.2f", p.Bright))
			cTolEntry.SetText(fmt.Sprint(p.Tol))
			cUnshade.Checked = p.Unshade
			cMirror.Checked = p.Mirror
			if selectTab != nil {
				selectTab("cube")
			}
			go refreshCube()
		}
	}

	presetList.OnSelected = func(id widget.ListItemID) {
		selectedPresetID = int(id)
		applyPreset(selectedPresetID)
	}
	presetList.OnUnselected = func(widget.ListItemID) {
		selectedPresetID = -1
	}
	presetRefresh := func() { presetList.Refresh() }

	applyBtn := widget.NewButton("Применить выбранный", func() {
		applyPreset(selectedPresetID)
	})
	savePresetBtn := widget.NewButton("Сохранить пресет (flat)", func() {
		dialog.ShowEntryDialog("Имя пресета", "", func(name string) {
			if strings.TrimSpace(name) == "" {
				return
			}
			ps := loadPresets()
			ps = append(ps, preset{
				Name: name, Mode: "flat",
				W: readInt(fWEntry, 16), H: readInt(fHEntry, 16), Colors: readInt(fColorsEntry, 4),
				Px: readInt(fPxEntry, 4), Nearest: fNearest.Checked, Bright: readFloat(fBrightEntry, 1),
			})
			savePresets(ps)
			presetRefresh()
		}, w)
	})
	saveCubePresetBtn := widget.NewButton("Сохранить cube-пресет", func() {
		dialog.ShowEntryDialog("Имя пресета", "", func(name string) {
			if strings.TrimSpace(name) == "" {
				return
			}
			ps := loadPresets()
			ps = append(ps, preset{
				Name: name, Mode: "cube", Face: cFace.Selected,
				Size: readInt(cSizeEntry, 16), Colors: readInt(cColorsEntry, 4), Px: readInt(cPxEntry, 4),
				SS: readInt(cSSEntry, 4), Inset: readFloat(cInsetEntry, 0.04),
				Bright: readFloat(cBrightEntry, 1), Tol: readInt(cTolEntry, 12),
				Unshade: cUnshade.Checked, Mirror: cMirror.Checked,
			})
			savePresets(ps)
			presetRefresh()
		}, w)
	})
	delPresetBtn := widget.NewButton("Удалить", func() {
		if selectedPresetID < 0 {
			return
		}
		ps := loadPresets()
		ps = append(ps[:selectedPresetID], ps[selectedPresetID+1:]...)
		savePresets(ps)
		selectedPresetID = -1
		presetRefresh()
	})

	presetPanel := container.NewBorder(
		container.NewPadded(widget.NewLabel("Пресеты хранятся в: "+presetsPath())),
		container.NewPadded(container.NewHBox(applyBtn, savePresetBtn, saveCubePresetBtn, delPresetBtn)),
		nil, nil,
		presetList,
	)

	flatTab := container.NewTabItem("Обычный режим", flatView)
	cubeTab := container.NewTabItem("Cube режим", cubeView)
	presetTab := container.NewTabItem("Пресеты", presetPanel)
	tabs := container.NewAppTabs(flatTab, cubeTab, presetTab)
	tabs.SetTabLocation(container.TabLocationLeading)

	selectTab = func(mode string) {
		if mode == "cube" {
			tabs.Select(cubeTab)
		} else {
			tabs.Select(flatTab)
		}
	}

	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}
		p := uriPath(uris[0])
		if tabs.Selected() == cubeTab {
			cSrc.SetText(p)
			go refreshCube()
		} else {
			fSrc.SetText(p)
			go refreshFlat()
		}
	})

	w.SetContent(tabs)
	w.ShowAndRun()
}

func loadImg(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}
