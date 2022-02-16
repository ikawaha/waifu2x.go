package engine

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Param represents parameters of a model.
type Param struct {
	Bias         []float64       `json:"bias"`         // バイアス
	KW           int             `json:"kW"`           // フィルタの幅
	KH           int             `json:"kH"`           // フィルタの高さ
	Weight       [][][][]float64 `json:"weight"`       // 重み
	NInputPlane  int             `json:"nInputPlane"`  // 入力平面数
	NOutputPlane int             `json:"nOutputPlane"` // 出力平面数
}

// Model represents a trained model.
type Model []Param

// LoadModelFile loads a trained model from the specified file.
func LoadModelFile(path string) (Model, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	return LoadModel(fp)
}

// LoadModel loads a trained model from the io.Reader.
func LoadModel(r io.Reader) (Model, error) {
	dec := json.NewDecoder(r)
	var m Model
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

//go:embed model/anime_style_art_rgb/* model/photo/*
var assets embed.FS

// LoadModelAssets loads a trained model from assets.
func LoadModelAssets(path string) (Model, error) {
	fsys, err := assets.Open(path)
	if err != nil {
		return nil, err
	}
	model, err := LoadModel(fsys)
	if err != nil {
		return nil, fmt.Errorf("failed to load : %w", err)
	}
	return model, nil
}

const (
	animeModelPath     = `model/anime_style_art_rgb/scale2.0x_model.json`
	animeNoisePathTmpl = `model/anime_style_art_rgb/noise%d_model.json`
	photoModelPath     = `model/photo/scale2.0x_model.json`
	photoNoisePathTmpl = `model/photo/noise%d_model.json`
)

// Mode is the type of trained models.
type Mode int

const (
	// Anime model type.
	Anime Mode = iota + 1
	// Photo model type.
	Photo
)

// String returns string representation of a mode.
func (t Mode) String() string {
	switch t {
	case Anime:
		return "anime"
	case Photo:
		return "photo"
	}
	return fmt.Sprintf("unknown type=%d", t)
}

// ModelSet is a set of trained models.
type ModelSet struct {
	Scale2xModel Model
	NoiseModel   Model
}

// NewAssetModelSet returns a set of trained models loaded from assets.
func NewAssetModelSet(t Mode, noiseLevel int) (*ModelSet, error) {
	if noiseLevel < 0 || noiseLevel > 3 {
		return nil, fmt.Errorf("invalid noise level: 0...3 but %d", noiseLevel)
	}
	var modelPath, noiseT string
	switch t {
	case Anime:
		modelPath, noiseT = animeModelPath, animeNoisePathTmpl
	case Photo:
		modelPath, noiseT = photoModelPath, photoNoisePathTmpl
	default:
		return nil, fmt.Errorf("unknown model type error")
	}
	var noise Model
	if noiseLevel > 0 {
		var err error
		noise, err = LoadModelAssets(fmt.Sprintf(noiseT, noiseLevel))
		if err != nil {
			return nil, fmt.Errorf("load noise model error: %w", err)
		}
	}
	scale, err := LoadModelAssets(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load scale model error: %w", err)
	}
	return &ModelSet{
		Scale2xModel: scale,
		NoiseModel:   noise,
	}, nil
}
