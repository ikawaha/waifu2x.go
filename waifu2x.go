package waifu2x

import (
	"fmt"
	"math"
)

type Waifu2x struct {
	Scale2xModel *Model
	NoiseModel   *Model
	Scale        float64
	IsDenoising  bool
}

func (w Waifu2x) Calc(pix []uint8, width, height int) ([]uint8, int, int) {
	if w.Scale2xModel == nil && w.NoiseModel == nil {
		return nil, 0, 0
	}

	// decompose
	r, g, b, a := channelDecompose(pix, width, height)
	//fmt.Printf("channelDecompose r:%d, g:%d, b:%d, a:%d\n",
	//	len(r.Buffer), len(g.Buffer), len(b.Buffer), len(a.Buffer))

	// de-noising
	if w.NoiseModel != nil {
		r, g, b = calcRGB(r, g, b, w.NoiseModel, 1)
		//fmt.Printf("noise r:%d, g:%d, b:%d, a:%d\n", len(r.Buffer), len(g.Buffer), len(b.Buffer), len(a.Buffer))
	}

	// calculate
	if w.Scale2xModel != nil {
		r, g, b = calcRGB(r, g, b, w.Scale2xModel, w.Scale)
		//fmt.Printf("scale r:%d, g:%d, b:%d, a:%d\n",
		//	len(r.Buffer), len(g.Buffer), len(b.Buffer), len(a.Buffer))
	}
	// resize alpha channel
	if w.Scale != 1 {
		a = a.resize(w.Scale)
	}

	if len(a.Buffer) != len(r.Buffer) {
		fmt.Printf("alpha:%d, red:%d\n", len(a.Buffer), len(r.Buffer))
		panic("A channel image size must be same with R channel image size")
	}

	// recompose
	//fmt.Printf("r:%d, g:%d, b:%d, a:%d\n", len(r.Buffer), len(g.Buffer), len(b.Buffer), len(a.Buffer))
	image2x, width, height := channelCompose(r, g, b, a)

	return image2x, width, height
}

func denormalize(p *ImagePlane) *ChannelImage {
	image := NewChannelImage(p.Width, p.Height)
	for i := 0; i < len(p.Buffer); i++ {
		v := int(math.Floor(p.getValueIndexed(i)*255.0) + 0.5)
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		image.Buffer[i] = uint8(v)
	}
	return image
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

func normalize(image *ChannelImage) *ImagePlane {
	width := image.Width
	height := image.Height
	imagePlane := NewImagePlane(width, height)
	if len(imagePlane.Buffer) != len(image.Buffer) {
		panic("Assertion error: length")
	}
	for i := 0; i < len(image.Buffer); i++ {
		imagePlane.setValueIndexed(i, float64(image.Buffer[i])/255.0)
	}
	return imagePlane
}

func typeW(model *Model) [][]float64 {
	if model == nil {
		panic("model nil")
	}
	var W [][]float64
	for l := 0; l < len(*model); l++ {
		// initialize weight matrix
		layerWeight := (*model)[l].Weight
		var vec []float64
		for _, d3 := range layerWeight {
			for _, d2 := range d3 {
				for _, w := range d2 {
					vec = append(vec, w...)
				}
			}
		}
		W = append(W, vec)
	}
	return W
}

func calcRGB(imageR, imageG, imageB *ChannelImage, model *Model, scale float64) (r, g, b *ChannelImage) {
	var inputPlanes []*ImagePlane
	for _, image := range []*ChannelImage{imageR, imageG, imageB} {
		imgResized := image
		if scale != 1.0 {
			imgResized = image.resize(scale)
		}
		imgExtra := imgResized.extrapolation(len(*model))
		inputPlanes = append(inputPlanes, normalize(imgExtra))
	}

	// blocking
	inputBlocks, blocksW, blocksH := blocking(inputPlanes)

	// init W
	W := typeW(model)

	outputBlocks := make([][]*ImagePlane, len(inputBlocks))
	for b := 0; b < len(inputBlocks); b++ {
		inputBlock := inputBlocks[b]
		var outputBlock []*ImagePlane
		for l := 0; l < len(*model); l++ {
			nOutputPlane := (*model)[l].NOutputPlane
			// convolution
			if model == nil {
				panic("xxx model nil")
			}
			outputBlock = convolution(inputBlock, W[l], nOutputPlane, (*model)[l].Bias)
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

	r = denormalize(outputPlanes[0])
	g = denormalize(outputPlanes[1])
	b = denormalize(outputPlanes[2])
	return
}
