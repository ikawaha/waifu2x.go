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

func Test_setWeightVec(t *testing.T) {
	model, err := LoadModelFile("./model/anime_style_art/scale2.0x_model.json")
	if err != nil {
		t.Fatalf("unexpected error, %v", err)
	}
	matrix := typeW(model)
	model.setWeightVec()
	for i, param := range model {
		for j, v := range param.WeightVec {
			if matrix[i][j] != v {
				t.Fatalf("[%d, %d]=%v <> %v", i, j, matrix[i][j], v)
			}
		}
	}
}

// W[][O*I*9]
func typeW(model Model) [][]float32 {
	var W [][]float32
	for l := range model {
		// initialize weight matrix
		param := model[l]
		var vec []float32
		// [nOutputPlane][nInputPlane][3][3]
		for i := 0; i < param.NInputPlane; i++ {
			for o := 0; o < param.NOutputPlane; o++ {
				vec = append(vec, param.Weight[o][i][0]...)
				vec = append(vec, param.Weight[o][i][1]...)
				vec = append(vec, param.Weight[o][i][2]...)
			}
		}
		W = append(W, vec)
	}
	return W
}
