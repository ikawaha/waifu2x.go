package model

import (
	"testing"
)

func TestLoadModel(t *testing.T) {
	files := []string{
		"./data/anime_style_art/noise1_model.json",
		"./data/anime_style_art/noise2_model.json",
		"./data/anime_style_art/noise3_model.json",
		"./data/anime_style_art/scale2.0x_model.json",
		"./data/anime_style_art_rgb/noise1_model.json",
		"./data/anime_style_art_rgb/noise2_model.json",
		"./data/anime_style_art_rgb/noise3_model.json",
		"./data/anime_style_art_rgb/scale2.0x_model.json",
		"./data/photo/noise1_model.json",
		"./data/photo/noise2_model.json",
		"./data/photo/noise3_model.json",
		"./data/photo/scale2.0x_model.json",
		"./data/ukbench/scale2.0x_model.json",
	}
	for _, f := range files {
		if _, err := LoadModelFile(f); err != nil {
			t.Errorf("unexpected error, %v", err)
		}
	}
}
