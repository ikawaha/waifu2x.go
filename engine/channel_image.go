package engine

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
)

// ChannelImage represents a discrete image.
type ChannelImage struct {
	Width  int
	Height int
	Buffer []uint8
}

// NewChannelImageWidthHeight returns a channel image of specific width and height.
func NewChannelImageWidthHeight(width, height int) ChannelImage {
	return ChannelImage{
		Width:  width,
		Height: height,
		Buffer: make([]uint8, width*height), // note. it is necessary to register all values less than 0 as 0 and greater than 255 as 255
	}
}

// NewChannelImage returns a channel image corresponding to the specified image.
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
	case *image.Paletted:
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

// NewDenormalizedChannelImage returns a channel image corresponding to the image plane.
func NewDenormalizedChannelImage(p ImagePlane) ChannelImage {
	img := NewChannelImageWidthHeight(p.Width, p.Height)
	for i := range p.Buffer {
		v := int(math.Round(p.Buffer[i] * 255.0))
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		img.Buffer[i] = uint8(v)
	}
	return img
}

// ImageRGBA converts the channel image to an image.RGBA and return it.
func (c ChannelImage) ImageRGBA() image.RGBA {
	r := image.Rect(0, 0, c.Width, c.Height)
	return image.RGBA{
		Pix:    c.Buffer,
		Stride: r.Dx() * 4,
		Rect:   r,
	}
}

// ImagePaletted converts the chanel image to an image.Paletted and return it.
func (c ChannelImage) ImagePaletted(p color.Palette) *image.Paletted {
	rgba := c.ImageRGBA()
	ret := image.NewPaletted(rgba.Bounds(), p)
	draw.DrawMask(ret, image.Rect(0, 0, ret.Bounds().Max.X, ret.Bounds().Max.Y), &rgba, image.Point{}, nil, image.Point{}, draw.Src)
	return ret
}

// ChannelDecompose decomposes a channel image to R, G, B and Alpha channels.
func ChannelDecompose(img ChannelImage) (r, g, b, a ChannelImage) {
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
	return r, g, b, a
}

// ChannelCompose composes R, G, B and Alpha channels to the one channel image.
func ChannelCompose(r, g, b, a ChannelImage) ChannelImage {
	width := r.Width
	height := r.Height
	img := make([]uint8, width*height*4)
	for i := 0; i < width*height; i++ {
		img[i*4] = r.Buffer[i]
		img[i*4+1] = g.Buffer[i]
		img[i*4+2] = b.Buffer[i]
		img[i*4+3] = a.Buffer[i]
	}
	return ChannelImage{
		Width:  width,
		Height: height,
		Buffer: img,
	}
}

// Extrapolation calculates an extrapolation algorithm.
func (c ChannelImage) Extrapolation(px int) ChannelImage {
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

// Resize returns a resized image.
func (c ChannelImage) Resize(scale float64) ChannelImage {
	if scale == 1.0 {
		return c
	}
	width := c.Width
	height := c.Height
	scaledWidth := int(math.Round(float64(width) * scale))
	scaledHeight := int(math.Round(float64(height) * scale))
	scaledImage := NewChannelImageWidthHeight(scaledWidth, scaledHeight)
	for w := 0; w < scaledWidth; w++ {
		for h := 0; h < scaledHeight; h++ {
			scaledIndex := w + (h * scaledWidth)
			wOriginal := int(math.Round(float64(w+1)/scale) - 1)
			if wOriginal < 0 {
				wOriginal = 0
			}
			hOriginal := int(math.Round(float64(h+1)/scale) - 1)
			if hOriginal < 0 {
				hOriginal = 0
			}
			indexOriginal := wOriginal + (hOriginal * width)
			scaledImage.Buffer[scaledIndex] = c.Buffer[indexOriginal]
		}
	}
	return scaledImage
}
