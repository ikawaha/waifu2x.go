package engine

import (
	"context"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Option represents an option of waifu2x.
type Option func(w *Waifu2x) error

// Parallel is the option that specifies the number of concurrency.
func Parallel(p int) Option {
	return func(w *Waifu2x) error {
		w.parallel = p
		return nil
	}
}

// Verbose is the verbose option.
func Verbose() Option {
	return func(w *Waifu2x) error {
		w.verbose = true
		return nil
	}
}

// Output is the option that sets the output destination.
func Output(w io.Writer) Option {
	return func(w2x *Waifu2x) error {
		w2x.output = w
		return nil
	}
}

// Waifu2x is the main structure for executing the waifu2x algorithm.
type Waifu2x struct {
	scaleModel Model
	noiseModel Model
	parallel   int
	verbose    bool
	output     io.Writer
}

// NewWaifu2x creates a Waifu2x structure.
func NewWaifu2x(mode Mode, noise int, opts ...Option) (*Waifu2x, error) {
	m, err := NewAssetModelSet(mode, noise)
	if err != nil {
		return nil, err
	}
	ret := &Waifu2x{
		scaleModel: m.Scale2xModel,
		noiseModel: m.NoiseModel,
		parallel:   1,
		output:     os.Stderr,
		verbose:    false,
	}
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (w Waifu2x) printf(format string, a ...interface{}) {
	if w.verbose {
		fmt.Fprintf(w.output, format, a...)
	}
}

func (w Waifu2x) println(a ...interface{}) {
	if w.verbose {
		fmt.Fprintln(w.output, a...)
	}
}

// ScaleUp scales up the image.
func (w Waifu2x) ScaleUp(ctx context.Context, img image.Image, scale float64) (image.RGBA, error) {
	ci, opaque, err := NewChannelImage(img)
	if err != nil {
		return image.RGBA{}, err
	}
	ci, err = w.convertChannelImage(ctx, ci, opaque, scale)
	return ci.ImageRGBA(), err
}

func (w Waifu2x) convertChannelImage(ctx context.Context, img ChannelImage, opaque bool, scale float64) (ChannelImage, error) {
	if (w.scaleModel == nil && w.noiseModel == nil) || scale <= 1 {
		return img, nil
	}

	w.printf("# of goroutines: %d\n", w.parallel)

	// decompose
	w.println("decomposing channels ...")
	r, g, b, a := ChannelDecompose(img)

	// de-noising
	if w.noiseModel != nil {
		w.println("de-noising ...")
		var err error
		r, g, b, err = w.convertRGB(ctx, r, g, b, w.noiseModel, 1, w.parallel)
		if err != nil {
			return ChannelImage{}, err
		}
	}

	// calculate
	if w.scaleModel != nil {
		w.println("scaling ...")
		var err error
		r, g, b, err = w.convertRGB(ctx, r, g, b, w.scaleModel, scale, w.parallel)
		if err != nil {
			return ChannelImage{}, err
		}
	}

	// alpha channel
	if !opaque {
		a = a.Resize(scale) // Resize simply
	} else if w.scaleModel != nil { // upscale the alpha channel
		w.println("scaling alpha ...")
		var err error
		a, _, _, err = w.convertRGB(ctx, a, a, a, w.scaleModel, scale, w.parallel)
		if err != nil {
			return ChannelImage{}, err
		}
	}

	if len(a.Buffer) != len(r.Buffer) {
		return ChannelImage{}, fmt.Errorf("channel image size must be same, A=%d, R=%d", len(a.Buffer), len(r.Buffer))
	}

	// recompose
	w.println("composing channels ...")
	return ChannelCompose(r, g, b, a), nil
}

func (w Waifu2x) convertRGB(ctx context.Context, imageR, imageG, imageB ChannelImage, model Model, scale float64, jobs int) (r, g, b ChannelImage, err error) {
	var inputPlanes [3]ImagePlane
	for i, img := range []ChannelImage{imageR, imageG, imageB} {
		imgResized := img.Resize(scale)
		imgExtra := imgResized.Extrapolation(len(model))
		p, err := NewNormalizedImagePlane(imgExtra)
		if err != nil {
			return ChannelImage{}, ChannelImage{}, ChannelImage{}, err
		}
		inputPlanes[i] = p
	}

	// Blocking
	inputBlocks, blocksW, blocksH := Blocking(inputPlanes)

	// init W
	W := typeW(model)

	inputLock := &sync.Mutex{}
	outputLock := &sync.Mutex{}
	sem := semaphore.NewWeighted(int64(jobs))
	wg := sync.WaitGroup{}
	outputBlocks := make([][]ImagePlane, len(inputBlocks))

	digits := int(math.Log10(float64(len(inputBlocks)))) + 2
	fmtStr := fmt.Sprintf("%%%dd/%%%dd", digits, digits) + " (%.1f%%)"

	w.printf(fmtStr, 0, len(inputBlocks), 0.0)

	for b := 0; b < len(inputBlocks); b++ {
		err := sem.Acquire(ctx, 1)
		if err != nil {
			panic(fmt.Sprintf("failed to acquire the semaphore: %s", err))
		}
		wg.Add(1)
		cb := b

		go func() {
			if cb >= 10 {
				w.printf("\x1b[2K\r"+fmtStr, cb+1, len(inputBlocks), float32(cb+1)/float32(len(inputBlocks))*100)
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
	w.println()
	inputBlocks = nil

	// de-blocking
	outputPlanes := Deblocking(outputBlocks, blocksW, blocksH)
	R := NewDenormalizedChannelImage(outputPlanes[0])
	G := NewDenormalizedChannelImage(outputPlanes[1])
	B := NewDenormalizedChannelImage(outputPlanes[2])
	return R, G, B, nil
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

func convolution(inputPlanes []ImagePlane, W []float64, nOutputPlane int, bias []float64) []ImagePlane {
	if len(inputPlanes) == 0 {
		return nil
	}
	width := inputPlanes[0].Width
	height := inputPlanes[0].Height
	outputPlanes := make([]ImagePlane, nOutputPlane)
	for i := 0; i < nOutputPlane; i++ {
		outputPlanes[i] = NewImagePlaneWidthHeight(width-2, height-2)
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
			for i := range inputPlanes {
				i00, i10, i20, i01, i11, i21, i02, i12, i22 := inputPlanes[i].SegmentAt(x, y)
				for o := 0; o < nOutputPlane; o++ {
					ws := W[wi : wi+9]
					sumValues[o] += ws[0]*i00 + ws[1]*i10 + ws[2]*i20 +
						ws[3]*i01 + ws[4]*i11 + ws[5]*i21 +
						ws[6]*i02 + ws[7]*i12 + ws[8]*i22
					wi += 9
				}
			}
			for o := 0; o < nOutputPlane; o++ {
				v := sumValues[o]
				if v < 0 {
					v *= 0.1
				}
				outputPlanes[o].SetAt(x-1, y-1, v)
			}
		}
	}
	return outputPlanes
}
