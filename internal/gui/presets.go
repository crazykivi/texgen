package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Preset struct {
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

func PresetsPath() string {
	dir, err := os.UserConfigDir()
	if err == nil {
		appDir := filepath.Join(dir, "texgen")
		os.MkdirAll(appDir, 0755)
		return filepath.Join(appDir, "presets.json")
	}
	return filepath.Join(".", "presets.json")
}

func LoadPresets() []Preset {
	b, err := os.ReadFile(PresetsPath())
	if err != nil {
		return nil
	}
	var ps []Preset
	_ = json.Unmarshal(b, &ps)
	out := make([]Preset, 0, len(ps))
	for _, p := range ps {
		if strings.TrimSpace(p.Name) == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func SavePresets(ps []Preset) {
	b, _ := json.MarshalIndent(ps, "", "  ")
	_ = os.WriteFile(PresetsPath(), b, 0644)
}
