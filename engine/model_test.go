package engine

import (
	"testing"
)

func TestLoadModel(t *testing.T) {
	files := []string{
		"./model/anime_style_art/noise1_model.json",
		"./model/anime_style_art/noise2_model.json",
		"./model/anime_style_art/noise3_model.json",
		"./model/anime_style_art/scale2.0x_model.json",
		"./model/anime_style_art_rgb/noise1_model.json",
		"./model/anime_style_art_rgb/noise2_model.json",
		"./model/anime_style_art_rgb/noise3_model.json",
		"./model/anime_style_art_rgb/scale2.0x_model.json",
		"./model/photo/noise1_model.json",
		"./model/photo/noise2_model.json",
		"./model/photo/noise3_model.json",
		"./model/photo/scale2.0x_model.json",
		"./model/ukbench/scale2.0x_model.json",
	}
	for _, f := range files {
		if _, err := LoadModelFile(f); err != nil {
			t.Errorf("unexpected error, %v", err)
		}
	}
}
