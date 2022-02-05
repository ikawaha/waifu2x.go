package waifu2x

import (
	"context"
	"fmt"
	"image"
	"math"
	"sync"

	"golang.org/x/sync/semaphore"
)

type Waifu2x struct {
	Scale2xModel *Model
	NoiseModel   *Model
	Scale        float64
	Jobs         int
}

func (w Waifu2x) Calc(pix []uint8, width, height int, enableAlphaUpscaling bool) ([]uint8, image.Rectangle) {
	if w.Scale2xModel == nil && w.NoiseModel == nil {
		return nil, image.Rectangle{}
	}

	fmt.Printf("# of goroutines: %d\n", w.Jobs)

	// decompose
	fmt.Println("decomposing channels ...")
	r, g, b, a := channelDecompose(pix, width, height)

	// de-noising
	if w.NoiseModel != nil {
		fmt.Println("de-noising ...")
		r, g, b = calcRGB(r, g, b, w.NoiseModel, 1, w.Jobs)
	}

	// calculate
	if w.Scale2xModel != nil {
		fmt.Println("upscaling ...")
		r, g, b = calcRGB(r, g, b, w.Scale2xModel, w.Scale, w.Jobs)
	}

	if enableAlphaUpscaling {
		// upscale the alpha channel
		if w.Scale2xModel != nil {
			fmt.Println("upscaling alpha ...")
			a, _, _ = calcRGB(a, a, a, w.Scale2xModel, w.Scale, w.Jobs)
		}
	} else {
		// resize the alpha channel simply
		if w.Scale != 1 {
			a = a.resize(w.Scale)
		}
	}

	if len(a.Buffer) != len(r.Buffer) {
		panic("A channel image size must be same with R channel image size")
	}

	// recompose
	fmt.Println("composing channels ...")
	image2x, width, height := channelCompose(r, g, b, a)

	return image2x, image.Rect(0, 0, width, height)
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

func calcRGB(imageR, imageG, imageB *ChannelImage, model *Model, scale float64, jobs int) (r, g, b *ChannelImage) {
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

	inputLock := &sync.Mutex{}
	outputLock := &sync.Mutex{}
	sem := semaphore.NewWeighted(int64(jobs))
	wg := sync.WaitGroup{}
	outputBlocks := make([][]*ImagePlane, len(inputBlocks))

	digits := int(math.Log10(float64(len(inputBlocks)))) + 2
	fmtStr := fmt.Sprintf("%%%dd/%%%dd", digits, digits) + " (%.1f%%)"

	fmt.Printf(fmtStr, 0, len(inputBlocks), 0.0)

	for b := 0; b < len(inputBlocks); b++ {
		err := sem.Acquire(context.TODO(), 1)
		if err != nil {
			panic(fmt.Sprintf("failed to acquire the semaphore: %s", err))
		}
		wg.Add(1)
		cb := b

		go func() {
			if cb >= 10 {
				fmt.Printf("\x1b[2K\r"+fmtStr, cb+1, len(inputBlocks), float32(cb+1)/float32(len(inputBlocks))*100)
			}

			inputBlock := inputBlocks[cb]
			var outputBlock []*ImagePlane
			for l := 0; l < len(*model); l++ {
				nOutputPlane := (*model)[l].NOutputPlane
				// convolution
				if model == nil {
					panic("xxx model nil")
				}
				outputBlock = convolution(inputBlock, W[l], nOutputPlane, (*model)[l].Bias)
				inputBlock = outputBlock // propagate output plane to next layer input

				inputLock.Lock()
				inputBlocks[cb] = nil
				inputLock.Unlock()
			}
			outputLock.Lock()
			outputBlocks[cb] = outputBlock
			outputLock.Unlock()
			sem.Release(1)
			wg.Done()
		}()
	}

	wg.Wait()
	fmt.Println()
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
