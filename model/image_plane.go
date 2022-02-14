package model

import (
	"fmt"
	"math"
)

const (
	BlockSize = 128
	Overlap   = 14
)

type ImagePlane struct {
	Width  int
	Height int
	Buffer []float64
}

func NewImagePlane(w, h int) ImagePlane {
	return ImagePlane{
		Width:  w,
		Height: h,
		Buffer: make([]float64, w*h),
	}
}

func (p ImagePlane) index(w, h int) int {
	return w + h*p.Width
}

func (p ImagePlane) getValue(w, h int) float64 {
	i := p.index(w, h)
	if i < 0 || i >= len(p.Buffer) {
		panic(fmt.Errorf("w %d, h %d, index %d, len(buf) %d", w, h, i, len(p.Buffer)))
	}
	// fmt.Printf("w %d, h %d, index %d, len(buf) %d\n", w, h, i, len(p.Buffer))
	return p.Buffer[i]
}

func (p ImagePlane) getBlock(x, y int) (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	i := (x - 1) + (y-1)*p.Width
	j := i + p.Width
	k := j + p.Width
	a := p.Buffer[i : i+3 : i+3]
	b := p.Buffer[j : j+3 : j+3]
	c := p.Buffer[k : k+3 : k+3]
	return a[0], a[1], a[2], b[0], b[1], b[2], c[0], c[1], c[2]
}

func (p *ImagePlane) setValue(w, h int, v float64) {
	p.Buffer[p.index(w, h)] = v
}

func (p ImagePlane) getValueIndexed(i int) float64 {
	return p.Buffer[i]
}

func (p *ImagePlane) setValueIndexed(i int, v float64) {
	p.Buffer[i] = v
}

func blocking(initialPlanes []ImagePlane) ([][]ImagePlane, int, int) {
	widthInput := initialPlanes[0].Width
	heightInput := initialPlanes[0].Height

	blocksW := int(math.Ceil(float64(widthInput-Overlap) / float64(BlockSize-Overlap)))
	blocksH := int(math.Ceil(float64(heightInput-Overlap) / float64(BlockSize-Overlap)))
	blocks := blocksW * blocksH

	// fmt.Println("BlockSize:", BlockSize)
	// fmt.Printf("blocksW:%d, blocksH:%d, blocks:%d\n", blocksW, blocksH, blocks)

	inputBlocks := make([][]ImagePlane, blocks)
	for b := 0; b < blocks; b++ {
		blockIndexW := b % blocksW
		blockIndexH := b / blocksW

		// fmt.Printf("blockIndexW:%d, blockIndexH:%d\n", blockIndexW, blockIndexH)

		blockWidth := BlockSize
		blockHeight := BlockSize

		if blockIndexW == blocksW-1 {
			blockWidth = widthInput - ((BlockSize - Overlap) * blockIndexW) // right end block
		}
		if blockIndexH == blocksH-1 {
			blockHeight = heightInput - ((BlockSize - Overlap) * blockIndexH) // bottom end block
		}

		// fmt.Printf("\t>>blockWidth:%d, blockHeight:%d\n", blockWidth, blockHeight)

		channels := make([]ImagePlane, len(initialPlanes))
		for i := 0; i < len(initialPlanes); i++ {
			channels[i] = NewImagePlane(blockWidth, blockHeight)
		}

		for w := 0; w < blockWidth; w++ {
			for h := 0; h < blockHeight; h++ {
				for i := 0; i < len(initialPlanes); i++ {
					targetIndexW := blockIndexW*(BlockSize-Overlap) + w
					targetIndexH := blockIndexH*(BlockSize-Overlap) + h
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

func deblocking(outputBlocks [][]ImagePlane, blocksW, blocksH int) []ImagePlane {
	blockSize := outputBlocks[0][0].Width
	var width int
	for b := 0; b < blocksW; b++ {
		width += outputBlocks[b][0].Width
	}

	var height int
	for b := 0; b < blocksW*blocksH; b += blocksW {
		height += outputBlocks[b][0].Height
	}

	outputPlanes := make([]ImagePlane, len(outputBlocks[0])) // XXX ???
	for b := 0; b < len(outputBlocks); b++ {
		block := outputBlocks[b]
		blockIndexW := b % blocksW
		blockIndexH := int(math.Floor(float64(b) / float64(blocksW)))

		for i := 0; i < len(block); i++ {
			if len(outputPlanes[i].Buffer) == 0 {
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
