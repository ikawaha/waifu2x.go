package waifu2x

import (
	"encoding/json"
	"io"
	"os"
)

type Layer struct {
	Bias         []float64       `json:"bias"`         //バイアス
	KW           int             `json:"kW"`           //フィルタの幅
	KH           int             `json:"kH"`           //フィルタの高さ
	NInputPlane  int             `json:"nInputPlane"`  //入力平面数
	NOutputPlane int             `json:"nOutputPlane"` //出力平面数
	Weight       [][][][]float64 `json:"weight"`       //重み
	WeightVec    []float64
}

type Model []Layer

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
	for i := range m {
		m[i].WeightVec = flattenWeight(m[i].Weight)
	}
	return m, nil
}

func (m Model) newPaddedPixels(p *Pixels) *Pixels {
	px := len(m)
	ex := NewPixels(p.Width+2*px, p.Height+2*px)
	for h := 0; h < p.Height+2*px; h++ {
		for w := 0; w < p.Width+2*px; w++ {
			index := w + h*(p.Width+2*px)
			if w < px {
				// Left outer area
				if h < px {
					// Left upper area
					ex.Pix[index] = p.Pix[0]
				} else if px+p.Height <= h {
					// Left lower area
					ex.Pix[index] = p.Pix[(p.Height-1)*p.Width]
				} else {
					// Left outer area
					ex.Pix[index] = p.Pix[(h-px)*p.Width]
				}
			} else if px+p.Width <= w {
				// Right outer area
				if h < px {
					// Right upper area
					ex.Pix[index] = p.Pix[p.Width-1]
				} else if px+p.Height <= h {
					// Right lower area
					ex.Pix[index] = p.Pix[p.Width-1+(p.Height-1)*p.Width]
				} else {
					// Right outer area
					ex.Pix[index] = p.Pix[p.Width-1+(h-px)*p.Width]
				}
			} else if h < px {
				// Upper outer area
				ex.Pix[index] = p.Pix[w-px]
			} else if px+p.Height <= h {
				// Lower outer area
				ex.Pix[index] = p.Pix[w-px+(p.Height-1)*p.Width]
			} else {
				// Inner area
				ex.Pix[index] = p.Pix[w-px+(h-px)*p.Width]
			}
		}
	}
	return ex
}

func (m Model) Encode(r, g, b *Pixels) (R, G, B *Pixels) {
	var inputPlanes []*ImagePlane
	for _, p := range []*Pixels{r, g, b} {
		p = m.newPaddedPixels(p)
		inputPlanes = append(inputPlanes, p.NewImagePlane())
	}

	// blocking
	inputBlocks, blocksW, blocksH := blocking(inputPlanes)

	outputBlocks := make([][]*ImagePlane, len(inputBlocks))
	for b := 0; b < len(inputBlocks); b++ {
		inputBlock := inputBlocks[b]
		var outputBlock []*ImagePlane
		for l := 0; l < len(m); l++ {
			nOutputPlane := m[l].NOutputPlane
			// convolution
			outputBlock = convolution(inputBlock, m[l].WeightVec, nOutputPlane, m[l].Bias)
			inputBlock = outputBlock // propagate output plane to next layer input
			inputBlocks[b] = nil
		}
		outputBlocks[b] = outputBlock
	}
	inputBlocks = nil

	// de-blocking
	outputPlanes := deblocking(outputBlocks, blocksW, blocksH)
	if len(outputPlanes) != 3 {
		panic("Output planes must be 3: color channel R, G, B.")
	}

	R = outputPlanes[0].NewPixels()
	G = outputPlanes[1].NewPixels()
	B = outputPlanes[2].NewPixels()
	return
}

func convolution(inputPlanes []*ImagePlane, W []float64, nOutputPlane int, bias []float64) []*ImagePlane {
	width := inputPlanes[0].Width
	height := inputPlanes[0].Height
	outputPlanes := make([]*ImagePlane, nOutputPlane)
	for i := 0; i < nOutputPlane; i++ {
		outputPlanes[i] = NewImagePlane(width-2, height-2)
	}
	sumValues := make([]float64, nOutputPlane)
	biasValues := make([]float64, nOutputPlane)
	for i := 0; i < nOutputPlane; i++ {
		biasValues[i] = bias[i]
	}
	for w := 1; w < width-1; w++ {
		for h := 1; h < height-1; h++ {
			for i := 0; i < len(biasValues); i++ {
				sumValues[i] = biasValues[i]
			}
			for i := 0; i < len(inputPlanes); i++ {
				i00 := inputPlanes[i].getValue(w-1, h-1)
				i10 := inputPlanes[i].getValue(w, h-1)
				i20 := inputPlanes[i].getValue(w+1, h-1)
				i01 := inputPlanes[i].getValue(w-1, h)
				i11 := inputPlanes[i].getValue(w, h)
				i21 := inputPlanes[i].getValue(w+1, h)
				i02 := inputPlanes[i].getValue(w-1, h+1)
				i12 := inputPlanes[i].getValue(w, h+1)
				i22 := inputPlanes[i].getValue(w+1, h+1)
				for o := 0; o < nOutputPlane; o++ {
					// assert inputPlanes.length == params.weight[o].length
					weight_index := (o * len(inputPlanes) * 9) + (i * 9)
					value := sumValues[o]
					value += i00 * W[weight_index]
					weight_index++
					value += i10 * W[weight_index]
					weight_index++
					value += i20 * W[weight_index]
					weight_index++
					value += i01 * W[weight_index]
					weight_index++
					value += i11 * W[weight_index]
					weight_index++
					value += i21 * W[weight_index]
					weight_index++
					value += i02 * W[weight_index]
					weight_index++
					value += i12 * W[weight_index]
					weight_index++
					value += i22 * W[weight_index]
					weight_index++
					sumValues[o] = value
				}
			}
			for o := 0; o < nOutputPlane; o++ {
				v := sumValues[o]
				//v += bias[o] // leaky ReLU bias is already added above
				if v < 0 {
					v *= 0.1
				}
				outputPlanes[o].setValue(w-1, h-1, v)
			}
		}
	}
	return outputPlanes
}
