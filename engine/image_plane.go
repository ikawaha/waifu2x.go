package engine

import (
	"fmt"
	"math"
)

const (
	// BlockSize is the size of a block when splitting the image plane.
	BlockSize = 128
	// Overlap is the size of the pixels that the image overlaps.
	Overlap = 14
)

// ImagePlane represents an image in which each pixel has a continuous value.
type ImagePlane struct {
	Width  int
	Height int
	Buffer []float64
}

// NewImagePlaneWidthHeight returns an image plane of specific width and height.
func NewImagePlaneWidthHeight(width, height int) ImagePlane {
	return ImagePlane{
		Width:  width,
		Height: height,
		Buffer: make([]float64, width*height),
	}
}

// NewNormalizedImagePlane create a normalized image plane from a channel image.
func NewNormalizedImagePlane(img ChannelImage) (ImagePlane, error) {
	p := NewImagePlaneWidthHeight(img.Width, img.Height)
	if len(img.Buffer) != len(p.Buffer) {
		return ImagePlane{}, fmt.Errorf("invalid image channel: width*heignt=%d <> len(buffer)=%d", img.Width*img.Height, img.Buffer)
	}
	for i := range img.Buffer {
		p.Buffer[i] = float64(img.Buffer[i]) / 255.0
	}
	return p, nil
}

// Index returns the buffer position corresponding to the specified width and height of the image.
func (p ImagePlane) Index(width, height int) int {
	return width + height*p.Width
}

// Value returns the value corresponding to the specified width and height of the image.
func (p ImagePlane) Value(width, height int) float64 {
	i := p.Index(width, height)
	if i < 0 || i >= len(p.Buffer) {
		panic(fmt.Errorf("width %d, height %d, Index %d, len(buf) %d", width, height, i, len(p.Buffer)))
	}
	// fmt.Printf("width %d, height %d, Index %d, len(buf) %d\n", width, height, i, len(p.Buffer))
	return p.Buffer[i]
}

// SegmentAt returns the 3x3 pixels at the specified position.
// [a0][a1][a2]
// [b0][b1][b2]
// [c0][c1][c2]   where (x, y) is b1.
func (p ImagePlane) SegmentAt(x, y int) (a0, a1, a2, b0, b1, b2, c0, c1, c2 float64) {
	i := (x - 1) + (y-1)*p.Width
	j := i + p.Width
	k := j + p.Width
	a := p.Buffer[i : i+3 : i+3]
	b := p.Buffer[j : j+3 : j+3]
	c := p.Buffer[k : k+3 : k+3]
	return a[0], a[1], a[2], b[0], b[1], b[2], c[0], c[1], c[2]
}

// SetAt sets the value to the buffer corresponding to the specified width and height of the image.
func (p *ImagePlane) SetAt(width, height int, v float64) {
	p.Buffer[p.Index(width, height)] = v
}

// Blocking divides a given image into blocks.
func Blocking(initialPlanes [3]ImagePlane) ([][]ImagePlane, int, int) {
	widthInput := initialPlanes[0].Width
	heightInput := initialPlanes[0].Height
	// blocks overlap 14px each other.
	blocksW := int(math.Ceil(float64(widthInput-Overlap) / float64(BlockSize-Overlap)))
	blocksH := int(math.Ceil(float64(heightInput-Overlap) / float64(BlockSize-Overlap)))
	blocks := blocksW * blocksH

	// fmt.Println("BlockSize:", BlockSize)
	// fmt.Printf("blocksW:%d, blocksH:%d, blocks:%d\n", blocksW, blocksH, blocks)

	inputBlocks := make([][]ImagePlane, blocks) // [ [ block0_R, block0_G, block0_B ], [ block1_R, ...] ... ]
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
		for i := range channels {
			channels[i] = NewImagePlaneWidthHeight(blockWidth, blockHeight)
		}

		for w := 0; w < blockWidth; w++ {
			for h := 0; h < blockHeight; h++ {
				for i := 0; i < len(initialPlanes); i++ {
					targetIndexW := blockIndexW*(BlockSize-Overlap) + w
					targetIndexH := blockIndexH*(BlockSize-Overlap) + h
					channel := initialPlanes[i]
					v := channel.Value(targetIndexW, targetIndexH)
					channels[i].SetAt(w, h, v)
				}
			}
		}
		inputBlocks[b] = channels
	}
	return inputBlocks, blocksW, blocksH
}

// Deblocking combines blocks for each of the R, G, and B channels.
func Deblocking(outputBlocks [][]ImagePlane, blocksW, blocksH int) [3]ImagePlane {
	blockSize := outputBlocks[0][0].Width
	var width int
	for b := 0; b < blocksW; b++ {
		width += outputBlocks[b][0].Width
	}

	var height int
	for b := 0; b < blocksW*blocksH; b += blocksW {
		height += outputBlocks[b][0].Height
	}

	var outputPlanes [3]ImagePlane // R,G,B
	for b := range outputBlocks {
		block := outputBlocks[b]
		blockIndexW := b % blocksW
		blockIndexH := int(math.Floor(float64(b) / float64(blocksW)))

		for i := 0; i < len(block); i++ {
			if len(outputPlanes[i].Buffer) == 0 {
				outputPlanes[i] = NewImagePlaneWidthHeight(width, height)
			}
			channelBlock := block[i]
			for w := 0; w < channelBlock.Width; w++ {
				for h := 0; h < channelBlock.Height; h++ {
					targetIndexW := blockIndexW*blockSize + w
					targetIndexH := blockIndexH*blockSize + h
					targetIndex := targetIndexH*width + targetIndexW
					v := channelBlock.Value(w, h)
					outputPlanes[i].Buffer[targetIndex] = v
				}
			}
		}
	}
	return outputPlanes
}
