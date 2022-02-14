package model

import (
	"context"
	"fmt"
	"math"
	"sync"

	"golang.org/x/sync/semaphore"
)

type Waifu2x struct {
	Scale2xModel Model
	NoiseModel   Model
	Scale        float64
	Jobs         int
}

func (w Waifu2x) Calc(img ChannelImage, enableAlphaUpscaling bool) ChannelImage {
	if w.Scale2xModel == nil && w.NoiseModel == nil {
		return ChannelImage{}
	}

	fmt.Printf("# of goroutines: %d\n", w.Jobs)

	// decompose
	fmt.Println("decomposing channels ...")
	r, g, b, a := channelDecompose(img)

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
	return channelCompose(r, g, b, a)
}

func denormalize(p ImagePlane) ChannelImage {
	img := NewChannelImage(p.Width, p.Height)
	for i := 0; i < len(p.Buffer); i++ {
		v := int(math.Floor(p.getValueIndexed(i)*255.0) + 0.5)
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		img.Buffer[i] = uint8(v)
	}
	return img
}

func convolution(inputPlanes []ImagePlane, W []float64, nOutputPlane int, bias []float64) []ImagePlane {
	width := inputPlanes[0].Width
	height := inputPlanes[0].Height
	outputPlanes := make([]ImagePlane, nOutputPlane)
	for i := 0; i < nOutputPlane; i++ {
		outputPlanes[i] = NewImagePlane(width-2, height-2)
	}
	sumValues := make([]float64, nOutputPlane)
	biasValues := make([]float64, nOutputPlane)
	for i := 0; i < nOutputPlane; i++ {
		biasValues[i] = bias[i]
	}
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			for i := 0; i < len(biasValues); i++ {
				sumValues[i] = biasValues[i]
			}
			wi := 0
			for i := 0; i < len(inputPlanes); i++ {
				i00, i10, i20, i01, i11, i21, i02, i12, i22 := inputPlanes[i].getBlock(x, y)
				for o := 0; o < nOutputPlane; o++ {
					ws := W[wi : wi+9]
					sumValues[o] += ws[0]*i00 + ws[1]*i10 + ws[2]*i20 + ws[3]*i01 + ws[4]*i11 + ws[5]*i21 + ws[6]*i02 + ws[7]*i12 + ws[8]*i22
					wi += 9
				}
			}
			for o := 0; o < nOutputPlane; o++ {
				v := sumValues[o]
				if v < 0 {
					v *= 0.1
				}
				outputPlanes[o].setValue(x-1, y-1, v)
			}
		}
	}
	return outputPlanes
}

func normalize(image ChannelImage) ImagePlane {
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

// W[][O*I*9]
func typeW(model Model) [][]float64 {
	var W [][]float64
	for l := 0; l < len(model); l++ {
		// initialize weight matrix
		param := model[l]
		var vec []float64
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

func calcRGB(imageR, imageG, imageB ChannelImage, model Model, scale float64, jobs int) (r, g, b ChannelImage) {
	var inputPlanes []ImagePlane
	for _, img := range []ChannelImage{imageR, imageG, imageB} {
		imgResized := img
		if scale != 1.0 {
			imgResized = img.resize(scale)
		}
		imgExtra := imgResized.extrapolation(len(model))
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
	outputBlocks := make([][]ImagePlane, len(inputBlocks))

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
			var outputBlock []ImagePlane
			for l := 0; l < len(model); l++ {
				nOutputPlane := model[l].NOutputPlane
				// convolution
				if model == nil {
					panic("xxx model nil")
				}
				outputBlock = convolution(inputBlock, W[l], nOutputPlane, model[l].Bias)
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
