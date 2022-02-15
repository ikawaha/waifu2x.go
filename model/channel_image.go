package model

import (
	"fmt"
	"image"
	"math"
)

type ChannelImage struct {
	Width  int
	Height int
	Buffer []uint8
}

func NewChannelImageWidthHeight(width, height int) ChannelImage {
	return ChannelImage{
		Width:  width,
		Height: height,
		Buffer: make([]uint8, width*height), // XXX 0以下を0, 255以上を255 として登録する必要あり
	}
}

func NewChannelImage(img image.Image) (ChannelImage, bool, error) {
	var (
		b      []uint8
		opaque bool
	)
	switch t := img.(type) {
	case *image.RGBA:
		b = t.Pix
		opaque = t.Opaque()
	case *image.NRGBA:
		b = t.Pix
		opaque = t.Opaque()
	case *image.YCbCr:
		r := t.Rect
		for y := 0; y < r.Dy(); y++ {
			for x := 0; x < r.Dx(); x++ {
				R, G, B, A := t.At(x, y).RGBA()
				b = append(b, uint8(R>>8), uint8(G>>8), uint8(B>>8), uint8(A>>8))
			}
		}
		opaque = t.Opaque()
	default:
		return ChannelImage{}, false, fmt.Errorf("unknown image format: %T", t)
	}
	return ChannelImage{
		Width:  img.Bounds().Max.X,
		Height: img.Bounds().Max.Y,
		Buffer: b,
	}, opaque, nil
}

func (c ChannelImage) ToRGBA() image.RGBA {
	r := image.Rect(0, 0, c.Width, c.Height)
	return image.RGBA{
		Pix:    c.Buffer,
		Stride: r.Dx() * 4,
		Rect:   r,
	}
}

func channelDecompose(img ChannelImage) (r, g, b, a ChannelImage) {
	r = NewChannelImageWidthHeight(img.Width, img.Height)
	g = NewChannelImageWidthHeight(img.Width, img.Height)
	b = NewChannelImageWidthHeight(img.Width, img.Height)
	a = NewChannelImageWidthHeight(img.Width, img.Height)
	for w := 0; w < img.Width; w++ {
		for h := 0; h < img.Height; h++ {
			i := w + h*img.Width
			r.Buffer[i] = img.Buffer[(w*4)+(h*img.Width*4)]
			g.Buffer[i] = img.Buffer[(w*4)+(h*img.Width*4)+1]
			b.Buffer[i] = img.Buffer[(w*4)+(h*img.Width*4)+2]
			a.Buffer[i] = img.Buffer[(w*4)+(h*img.Width*4)+3]
		}
	}
	return
}

func channelCompose(imageR, imageG, imageB, imageA ChannelImage) ChannelImage {
	width := imageR.Width
	height := imageR.Height
	img := make([]uint8, width*height*4)
	if width*height != len(imageR.Buffer) {
		panic(fmt.Errorf("channelCompose() buflen:%d, width*height:%d", len(imageR.Buffer), width*height))
	}
	for i := 0; i < width*height; i++ {
		img[i*4] = imageR.Buffer[i]
		img[i*4+1] = imageG.Buffer[i]
		img[i*4+2] = imageB.Buffer[i]
		img[i*4+3] = imageA.Buffer[i]
	}
	return ChannelImage{
		Width:  width,
		Height: height,
		Buffer: img,
	}
}

func (c ChannelImage) extrapolation(px int) ChannelImage {
	width := c.Width
	height := c.Height
	toIndex := func(w, h int) int {
		return w + h*width
	}
	imageEx := NewChannelImageWidthHeight(width+(2*px), height+(2*px))
	for h := 0; h < height+(px*2); h++ {
		for w := 0; w < width+(px*2); w++ {
			index := w + h*(width+(px*2))
			if w < px {
				// Left outer area
				if h < px {
					// Left upper area
					imageEx.Buffer[index] = c.Buffer[toIndex(0, 0)]
				} else if px+height <= h {
					// Left lower area
					imageEx.Buffer[index] = c.Buffer[toIndex(0, height-1)]
				} else {
					// Left outer area
					imageEx.Buffer[index] = c.Buffer[toIndex(0, h-px)]
				}
			} else if px+width <= w {
				// Right outer area
				if h < px {
					// Right upper area
					imageEx.Buffer[index] = c.Buffer[toIndex(width-1, 0)]
				} else if px+height <= h {
					// Right lower area
					imageEx.Buffer[index] = c.Buffer[toIndex(width-1, height-1)]
				} else {
					// Right outer area
					imageEx.Buffer[index] = c.Buffer[toIndex(width-1, h-px)]
				}
			} else if h < px {
				// Upper outer area
				imageEx.Buffer[index] = c.Buffer[toIndex(w-px, 0)]
			} else if px+height <= h {
				// Lower outer area
				imageEx.Buffer[index] = c.Buffer[toIndex(w-px, height-1)]
			} else {
				// Inner area
				imageEx.Buffer[index] = c.Buffer[toIndex(w-px, h-px)]
			}
		}
	}
	return imageEx
}

func (c ChannelImage) resize(scale float64) ChannelImage {
	width := c.Width
	height := c.Height
	scaledWidth := int(math.Floor(float64(width)*scale + 0.5))   // Round
	scaledHeight := int(math.Floor(float64(height)*scale + 0.5)) // Round
	scaledImage := NewChannelImageWidthHeight(scaledWidth, scaledHeight)
	for w := 0; w < scaledWidth; w++ {
		for h := 0; h < scaledHeight; h++ {
			scaledIndex := w + (h * scaledWidth)
			wOriginal := int(math.Floor((float64(w+1)/scale)+0.5) - 1) // Round
			if wOriginal < 0 {
				wOriginal = 0
			}
			hOriginal := int(math.Floor((float64(h+1)/scale)+0.5) - 1) // Round
			if hOriginal < 0 {
				hOriginal = 0
			}
			indexOriginal := wOriginal + (hOriginal * width)
			scaledImage.Buffer[scaledIndex] = c.Buffer[indexOriginal]
		}
	}
	return scaledImage
}
