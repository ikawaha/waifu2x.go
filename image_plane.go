package waifu2x

import (
	"fmt"
	"math"
)

const (
	blockSize = 128
	overlap   = 14
)

// ImagePlane represents the image plane.
type ImagePlane struct {
	Width  int
	Height int
	Pix    []float64
}

// NewImagePlane creates the image plane.
func NewImagePlane(w, h int) *ImagePlane {
	return &ImagePlane{
		Width:  w,
		Height: h,
		Pix:    make([]float64, w*h),
	}
}

func (p ImagePlane) getLength() int {
	return len(p.Pix)
}

func (p ImagePlane) getPix() []float64 {
	return p.Pix
}

func (p ImagePlane) index(w, h int) int {
	return w + h*p.Width
}

func (p ImagePlane) getValue(w, h int) float64 {
	i := p.index(w, h)
	if i < 0 || i >= len(p.Pix) {
		panic(fmt.Errorf("w %d, h %d, index %d, len(buf) %d", w, h, i, len(p.Pix)))
	}
	//fmt.Printf("w %d, h %d, index %d, len(buf) %d\n", w, h, i, len(p.Pix))
	return p.Pix[i]
}

func (p *ImagePlane) setValue(w, h int, v float64) {
	p.Pix[p.index(w, h)] = v
}

func (p *ImagePlane) getValueIndexed(i int) float64 {
	return p.Pix[i]
}

func (p *ImagePlane) setValueIndexed(i int, v float64) {
	p.Pix[i] = v
}

// Stream represents the divided image channels.
type Stream []*ImagePlane

//type Stream struct {
//	ID       int
//	Channels []*ImagePlane
//}

func divide(initialPlanes []*ImagePlane) (out []Stream, cols, rows int) {
	widthInput := initialPlanes[0].Width
	heightInput := initialPlanes[0].Height

	blocksW := int(math.Ceil(float64(widthInput-overlap) / float64(blockSize-overlap)))
	blocksH := int(math.Ceil(float64(heightInput-overlap) / float64(blockSize-overlap)))
	blocks := blocksW * blocksH

	inputBlocks := make([]Stream, blocks)
	for b := 0; b < blocks; b++ {
		blockIndexW := b % blocksW
		blockIndexH := b / blocksW

		blockWidth := blockSize
		blockHeight := blockSize

		if blockIndexW == blocksW-1 {
			blockWidth = widthInput - ((blockSize - overlap) * blockIndexW) // right end block
		}
		if blockIndexH == blocksH-1 {
			blockHeight = heightInput - ((blockSize - overlap) * blockIndexH) // bottom end block
		}

		channels := make(Stream, len(initialPlanes))
		for i := 0; i < len(initialPlanes); i++ {
			channels[i] = NewImagePlane(blockWidth, blockHeight)
		}

		for w := 0; w < blockWidth; w++ {
			for h := 0; h < blockHeight; h++ {
				for i := 0; i < len(initialPlanes); i++ {
					targetIndexW := blockIndexW*(blockSize-overlap) + w
					targetIndexH := blockIndexH*(blockSize-overlap) + h
					channel := initialPlanes[i]
					v := channel.getValue(targetIndexW, targetIndexH)
					channels[i].setValue(w, h, v)
				}
			}
		}
		inputBlocks[b] = channels
	}
	return inputBlocks, blocksW, blocksH
}

func conquer(outputBlocks []Stream, blocksW, blocksH int) []*ImagePlane {
	blockSize := outputBlocks[0][0].Width
	var width int
	for b := 0; b < blocksW; b++ {
		width += outputBlocks[b][0].Width
	}

	var height int
	for b := 0; b < blocksW*blocksH; b += blocksW {
		height += outputBlocks[b][0].Height
	}

	outputPlanes := make([]*ImagePlane, len(outputBlocks[0])) //XXX ???
	for b := 0; b < len(outputBlocks); b++ {
		block := outputBlocks[b]
		blockIndexW := b % blocksW
		blockIndexH := int(math.Floor(float64(b) / float64(blocksW)))

		for i := 0; i < len(block); i++ {
			if outputPlanes[i] == nil {
				outputPlanes[i] = NewImagePlane(width, height)
			}
			channelBlock := block[i]
			for w := 0; w < channelBlock.Width; w++ {
				for h := 0; h < channelBlock.Height; h++ {
					targetIndexW := blockIndexW*blockSize + w
					targetIndexH := blockIndexH*blockSize + h
					targetIndex := targetIndexH*width + targetIndexW
					v := channelBlock.getValue(w, h)
					outputPlanes[i].setValueIndexed(targetIndex, v)
				}
			}
		}
	}
	return outputPlanes
}

// NewPixels create the image from the image plane.
func (p ImagePlane) NewPixels() *Pixels {
	pix := NewPixels(p.Width, p.Height)
	for i := 0; i < len(p.Pix); i++ {
		v := math.Floor(p.Pix[i]*255.0) + 0.5
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		pix.Pix[i] = uint8(v)
	}
	return pix
}
