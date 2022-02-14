package model

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Param struct {
	Bias         []float64       `json:"bias"`         // バイアス
	KW           int             `json:"kW"`           // フィルタの幅
	KH           int             `json:"kH"`           // フィルタの高さ
	Weight       [][][][]float64 `json:"weight"`       // 重み
	NInputPlane  int             `json:"nInputPlane"`  // 入力平面数
	NOutputPlane int             `json:"nOutputPlane"` // 出力平面数
}

type Model []Param

func LoadModelFile(path string) (Model, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	return LoadModel(fp)
}

func LoadModel(r io.Reader) (Model, error) {
	dec := json.NewDecoder(r)
	var m Model
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

//go:embed data/anime_style_art_rgb/* data/photo/*
var assets embed.FS

func LoadModelFromAssets(path string) (Model, error) {
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
	animeModelPath     = `data/anime_style_art_rgb/scale2.0x_model.json`
	animeNoisePathTmpl = `data/anime_style_art_rgb/noise%d_model.json`
	photoModelPath     = `data/photo/scale2.0x_model.json`
	photoNoisePathTmpl = `data/photo/noise%d_model.json`
)

type ModelType int

const (
	Anime ModelType = iota + 1
	Photo
)

func (t ModelType) String() string {
	switch t {
	case Anime:
		return "anime"
	case Photo:
		return "photo"
	}
	return fmt.Sprintf("unknown type=%d", t)
}

type ModelSet struct {
	Scale2xModel Model
	NoiseModel   Model
}

func newAssetModelSet(t ModelType, noiseLevel int) (*ModelSet, error) {
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
		noise, err = LoadModelFromAssets(fmt.Sprintf(noiseT, noiseLevel))
		if err != nil {
			return nil, fmt.Errorf("load noise model error: %w", err)
		}
	}
	scale, err := LoadModelFromAssets(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load scale model error: %w", err)
	}
	return &ModelSet{
		Scale2xModel: scale,
		NoiseModel:   noise,
	}, nil
}

func NewAnimeModelSet(noiseLevel int) (*ModelSet, error) {
	return newAssetModelSet(Anime, noiseLevel)
}

func NewPhotoModelSet(noiseLevel int) (*ModelSet, error) {
	return newAssetModelSet(Photo, noiseLevel)
}
