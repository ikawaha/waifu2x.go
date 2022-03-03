package engine

import (
	"context"
	"fmt"
	"image"
	"image/gif"
	"io"
	"math"
	"os"
	"runtime"
	"sync"
)

// Option represents an option of waifu2x.
type Option func(w *Waifu2x) error

// Parallel sets the option that specifies the limit number of concurrency.
func Parallel(p int) Option {
	return func(w *Waifu2x) error {
		if p < 0 {
			return fmt.Errorf("an integer value less than 1")
		}
		w.parallel = p
		return nil
	}
}

// Verbose sets the verbose option.
func Verbose(v bool) Option {
	return func(w *Waifu2x) error {
		w.verbose = v
		return nil
	}
}

// LogOutput sets the log output destination.
func LogOutput(w io.Writer) Option {
	return func(w2x *Waifu2x) error {
		w2x.logOutput = w
		return nil
	}
}

// Waifu2x is the main structure for executing the waifu2x algorithm.
type Waifu2x struct {
	scaleModel Model
	noiseModel Model
	parallel   int
	verbose    bool
	logOutput  io.Writer
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
		logOutput:  os.Stderr,
		parallel:   runtime.GOMAXPROCS(runtime.NumCPU()),
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
		fmt.Fprintf(w.logOutput, format, a...)
	}
}

func (w Waifu2x) println(a ...interface{}) {
	if w.verbose {
		fmt.Fprintln(w.logOutput, a...)
	}
}

func (w Waifu2x) ScaleUpGIF(ctx context.Context, img *gif.GIF, scale float64) (*gif.GIF, error) {
	frames := make([]*image.Paletted, 0, len(img.Image))
	for _, v := range img.Image {
		p := v.Palette
		ci, err := w.ScaleUp(ctx, v, scale)
		if err != nil {
			return nil, err
		}
		ip := ci.ImagePaletted(p)
		frames = append(frames, ip)
	}
	img.Image = frames
	img.Config.Width = int(float64(img.Config.Width) * scale)
	img.Config.Height = int(float64(img.Config.Height) * scale)
	return img, nil
}

// ScaleUp scales up the image.
func (w Waifu2x) ScaleUp(ctx context.Context, img image.Image, scale float64) (ChannelImage, error) {
	ci, _, err := NewChannelImage(img)
	if err != nil {
		return ChannelImage{}, err
	}
	for {
		if scale < 2.0 {
			ci, err = w.convertChannelImage(ctx, ci, scale)
			if err != nil {
				return ChannelImage{}, err
			}
			break
		}
		ci, err = w.convertChannelImage(ctx, ci, 2)
		if err != nil {
			return ChannelImage{}, err
		}
		scale = scale / 2.0
	}
	return ci, err
}

func (w Waifu2x) convertChannelImage(ctx context.Context, img ChannelImage, scale float64) (ChannelImage, error) {
	if (w.scaleModel == nil && w.noiseModel == nil) || scale <= 1 {
		return img, nil
	}

	if w.parallel > 0 {
		w.printf("# of goroutines: %d\n", w.parallel)
	}

	// decompose
	w.println("decomposing channels ...")
	r, g, b, a := ChannelDecompose(img)

	// de-noising
	if w.noiseModel != nil {
		w.println("de-noising ...")
		var err error
		r, g, b, err = w.convertRGB(ctx, r, g, b, w.noiseModel, 1)
		if err != nil {
			return ChannelImage{}, err
		}
	}

	// calculate
	if w.scaleModel != nil {
		w.println("scaling ...")
		var err error
		r, g, b, err = w.convertRGB(ctx, r, g, b, w.scaleModel, scale)
		if err != nil {
			return ChannelImage{}, err
		}
	}

	// alpha channel
	a = a.Resize(scale)
	/*
		if !opaque {
			a = a.Resize(scale) // Resize simply
		} else if w.scaleModel != nil { // upscale the alpha channel
			w.println("scaling alpha ...")
			var err error
			a, _, _, err = w.convertRGB(ctx, a, a, a, w.scaleModel, scale)
			if err != nil {
				return ChannelImage{}, err
			}
		}
	*/

	if len(a.Buffer) != len(r.Buffer) {
		return ChannelImage{}, fmt.Errorf("channel image size must be same, A=%d, R=%d", len(a.Buffer), len(r.Buffer))
	}

	// recompose
	w.println("composing channels ...")
	return ChannelCompose(r, g, b, a), nil
}

func (w Waifu2x) convertRGB(_ context.Context, imageR, imageG, imageB ChannelImage, model Model, scale float64) (r, g, b ChannelImage, err error) {
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
	outputBlocks := make([][]ImagePlane, len(inputBlocks))

	digits := int(math.Log10(float64(len(inputBlocks)))) + 2
	fmtStr := fmt.Sprintf("%%%dd/%%%dd", digits, digits) + " (%.1f%%)"
	w.printf(fmtStr, 0, len(inputBlocks), 0.0)

	limit := make(chan struct{}, w.parallel)
	wg := sync.WaitGroup{}
	for i := range inputBlocks {
		wg.Add(1)
		go func(i int) {
			limit <- struct{}{}
			defer wg.Done()
			if i >= 10 {
				w.printf("\x1b[2K\r"+fmtStr, i+1, len(inputBlocks), float32(i+1)/float32(len(inputBlocks))*100)
			}
			inputBlock := inputBlocks[i]
			var outputBlock []ImagePlane
			for l := range model {
				nOutputPlane := model[l].NOutputPlane
				// convolution
				outputBlock = convolution(inputBlock, model[l].WeightVec, nOutputPlane, model[l].Bias)
				inputBlock = outputBlock // propagate output plane to next layer input
				inputBlocks[i] = nil
			}
			outputBlocks[i] = outputBlock
			<-limit
		}(i)
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
			for i := range biasValues {
				sumValues[i] = biasValues[i]
			}
			const square = 9
			wi := 0
			for i := range inputPlanes {
				a0, a1, a2, b0, b1, b2, c0, c1, c2 := inputPlanes[i].SegmentAt(x, y)
				for o := 0; o < nOutputPlane; o++ {
					ws := W[wi : wi+square] // 3x3 square
					sumValues[o] = sumValues[o] +
						ws[0]*a0 + ws[1]*a1 + ws[2]*a2 +
						ws[3]*b0 + ws[4]*b1 + ws[5]*b2 +
						ws[6]*c0 + ws[7]*c1 + ws[8]*c2
					wi += square
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
