package waifu2x

import (
	"encoding/json"
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

func LoadModelFile(path string) (*Model, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	return LoadModel(fp)
}

func LoadModel(r io.Reader) (*Model, error) {
	dec := json.NewDecoder(r)
	var m Model
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}
