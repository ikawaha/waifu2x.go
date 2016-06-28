package waifu2x

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

const (
	// AnimeStyleArtNoise1ModelFilePath : models/anime_style_art/noise1_model.json
	AnimeStyleArtNoise1ModelFilePath = "models/anime_style_art/noise1_model.json"

	// AnimeStyleArtNoise2ModelFilePath : models/anime_style_art/noise2_model.json
	AnimeStyleArtNoise2ModelFilePath = "models/anime_style_art/noise2_model.json"

	// AnimeStyleArtNoise3ModelFilePath : models/anime_style_art/noise3_model.json
	AnimeStyleArtNoise3ModelFilePath = "models/anime_style_art/noise3_model.json"

	// AnimeStyleArtScale2xModelFilePath : models/anime_style_art/scale2.0x_model.json
	AnimeStyleArtScale2xModelFilePath = "models/anime_style_art/scale2.0x_model.json"

	// AnimeStyleArtRGBNoise1ModelFilePath : models/anime_style_art_rgb/noise1_model.json
	AnimeStyleArtRGBNoise1ModelFilePath = "models/anime_style_art_rgb/noise1_model.json"

	// AnimeStyleArtRGBNoise2ModelFilePath : models/anime_style_art_rgb/noise2_model.json
	AnimeStyleArtRGBNoise2ModelFilePath = "models/anime_style_art_rgb/noise2_model.json"

	// AnimeStyleArtRGBNoise3ModelFilePath : models/anime_style_art_rgb/noise3_model.json
	AnimeStyleArtRGBNoise3ModelFilePath = "models/anime_style_art_rgb/noise3_model.json"

	// AnimeStyleArtRGBScale2xModelFilePath : models/anime_style_art_rgb/scale2.0x_model.json
	AnimeStyleArtRGBScale2xModelFilePath = "models/anime_style_art_rgb/scale2.0x_model.json"

	// PhotoNoise1ModelFilePath : models/photo/noise1_model.json
	PhotoNoise1ModelFilePath = "models/photo/noise1_model.json"

	// PhotoNoise2ModelFilePath : models/photo/noise2_model.json
	PhotoNoise2ModelFilePath = "models/photo/noise2_model.json"

	// PhotoNoise3ModelFilePath : models/photo/noise3_model.json
	PhotoNoise3ModelFilePath = "models/photo/noise3_model.json"

	// PhotoScale2xModelFilePath : models/photo/scale2.0x_model.json
	PhotoScale2xModelFilePath = "models/photo/scale2.0x_model.json"

	// UkbenchScale2xModelFilePath : models/ukbench/scale2.0x_model.json
	UkbenchScale2xModelFilePath = "models/ukbench/scale2.0x_model.json"
)

// Model represents a model of a Deep Learning
type Model []Layer

// Layer represents a layer of a model
type Layer struct {
	Bias         []float64       `json:"bias"`         //バイアス
	KW           int             `json:"kW"`           //フィルタの幅
	KH           int             `json:"kH"`           //フィルタの高さ
	NInputPlane  int             `json:"nInputPlane"`  //入力平面数
	NOutputPlane int             `json:"nOutputPlane"` //出力平面数
	Weight       [][][][]float64 `json:"weight"`       //重み
	WeightVec    []float64
}

func flattenWeight(weight [][][][]float64) []float64 {
	var vec []float64
	for _, v3 := range weight {
		for _, v2 := range v3 {
			for _, v1 := range v2 {
				vec = append(vec, v1...)
			}
		}
	}
	return vec
}

// LoadModelFile loads a model from a json file
func LoadModelFile(path string) (Model, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	return LoadModel(fp)
}

// LoadModel loads a model
func LoadModel(r io.Reader) (Model, error) {
	dec := json.NewDecoder(r)
	var m Model
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	for i := range m {
		m[i].WeightVec = flattenWeight(m[i].Weight)
	}
	return m, nil
}

func (m Model) newPaddedPixels(p *Pixels) *Pixels {
	px := len(m)
	ex := NewPixels(p.Width+2*px, p.Height+2*px)
	for h := 0; h < ex.Height; h++ {
		for w := 0; w < ex.Width; w++ {
			i := w + h*ex.Width
			if w < px { // Left outer area
				if h < px {
					ex.Pix[i] = p.Pix[0] // Left upper area
				} else if px+p.Height <= h {
					ex.Pix[i] = p.Pix[(p.Height-1)*p.Width] // Left lower area
				} else {
					ex.Pix[i] = p.Pix[(h-px)*p.Width] // Left outer area

				}
			} else if px+p.Width <= w { // Right outer area
				if h < px {
					ex.Pix[i] = p.Pix[p.Width-1] // Right upper area
				} else if px+p.Height <= h {
					ex.Pix[i] = p.Pix[p.Width-1+(p.Height-1)*p.Width] // Right lower area
				} else {
					ex.Pix[i] = p.Pix[p.Width-1+(h-px)*p.Width] // Right outer area
				}
			} else if h < px {
				ex.Pix[i] = p.Pix[w-px] // Upper outer area
			} else if px+p.Height <= h {
				ex.Pix[i] = p.Pix[w-px+(p.Height-1)*p.Width] // Lower outer area
			} else {
				ex.Pix[i] = p.Pix[w-px+(h-px)*p.Width] // Inner area
			}
		}
	}
	return ex
}

// Encode applies model to RGB images
func (m Model) Encode(r, g, b *Pixels) (R, G, B *Pixels) {
	channels := make([]*ImagePlane, 0, 3)
	for _, p := range []*Pixels{r, g, b} {
		p = m.newPaddedPixels(p)
		channels = append(channels, p.NewImagePlane())
	}

	inputs, cols, rows := divide(channels)
	outputs := make([]Stream, len(inputs))
	var wg sync.WaitGroup
	for i := range inputs {
		wg.Add(1)
		go func(in Stream, i int) {
			defer wg.Done()
			var out Stream
			for _, layer := range m {
				out = layer.convolution(in)
				in = out
			}
			outputs[i] = out
		}(inputs[i], i)
	}
	wg.Wait()
	channels = conquer(outputs, cols, rows)

	if len(channels) != 3 {
		panic("Output planes must be 3: color channel R, G, B.") //XXX
	}

	R = channels[0].NewPixels()
	G = channels[1].NewPixels()
	B = channels[2].NewPixels()
	return
}

func (l Layer) convolution(input Stream) Stream {

	W := l.WeightVec

	width := input[0].Width
	height := input[0].Height
	output := make(Stream, l.NOutputPlane)
	for i := range output {
		output[i] = NewImagePlane(width-2, height-2)
	}

	biasValues := make([]float64, l.NOutputPlane)
	for i := range biasValues {
		biasValues[i] = l.Bias[i]
	}
	sumValues := make([]float64, l.NOutputPlane)
	for w := 1; w < width-1; w++ {
		for h := 1; h < height-1; h++ {
			for i := 0; i < len(biasValues); i++ {
				sumValues[i] = biasValues[i]
			}
			for i := 0; i < len(input); i++ {
				i00 := input[i].getValue(w-1, h-1)
				i10 := input[i].getValue(w, h-1)
				i20 := input[i].getValue(w+1, h-1)
				i01 := input[i].getValue(w-1, h)
				i11 := input[i].getValue(w, h)
				i21 := input[i].getValue(w+1, h)
				i02 := input[i].getValue(w-1, h+1)
				i12 := input[i].getValue(w, h+1)
				i22 := input[i].getValue(w+1, h+1)
				for o := 0; o < l.NOutputPlane; o++ {
					idx := (o * len(input) * 9) + (i * 9)
					value := sumValues[o]
					value += i00 * W[idx]
					idx++
					value += i10 * W[idx]
					idx++
					value += i20 * W[idx]
					idx++
					value += i01 * W[idx]
					idx++
					value += i11 * W[idx]
					idx++
					value += i21 * W[idx]
					idx++
					value += i02 * W[idx]
					idx++
					value += i12 * W[idx]
					idx++
					value += i22 * W[idx]
					idx++
					sumValues[o] = value
				}
			}
			for o := 0; o < l.NOutputPlane; o++ {
				v := sumValues[o]
				if v < 0 {
					v *= 0.1
				}
				output[o].setValue(w-1, h-1, v)
			}
		}
	}
	return output
}
