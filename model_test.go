package waifu2x

import (
	"reflect"
	"testing"
)

func TestLoadModel(t *testing.T) {
	files := []string{
		"./models/anime_style_art/noise1_model.json",
		"./models/anime_style_art/noise2_model.json",
		"./models/anime_style_art/noise3_model.json",
		"./models/anime_style_art/scale2.0x_model.json",
		"./models/anime_style_art_rgb/noise1_model.json",
		"./models/anime_style_art_rgb/noise2_model.json",
		"./models/anime_style_art_rgb/noise3_model.json",
		"./models/anime_style_art_rgb/scale2.0x_model.json",
		"./models/photo/noise1_model.json",
		"./models/photo/noise2_model.json",
		"./models/photo/noise3_model.json",
		"./models/photo/scale2.0x_model.json",
		"./models/ukbench/scale2.0x_model.json",
	}
	for _, f := range files {
		if _, err := LoadModelFile(f); err != nil {
			t.Errorf("unexpected error, %v", err)
		}
	}
}

func TestFlattenWeight(t *testing.T) {
	vec := flattenWeight([][][][]float64{{{{0, 1, 2, 3, 4}}}})
	expected := []float64{0, 1, 2, 3, 4}
	if !reflect.DeepEqual(expected, vec) {
		t.Errorf("got %v, expected %+v", vec, expected)
	}
}
