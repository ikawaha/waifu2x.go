package waifu2x

import (
	"fmt"
	"math"
)

// Pixels represents the image.
type Pixels struct {
	Width  int
	Height int
	Pix    []uint8 // 0以下を0, 255以上を255 とする
}

// NewPixels creates the image.
func NewPixels(width, height int) *Pixels {
	return &Pixels{
		Width:  width,
		Height: height,
		Pix:    make([]uint8, width*height),
	}
}

// NewImagePlane creates the image plane.
func (p Pixels) NewImagePlane() *ImagePlane {
	pl := NewImagePlane(p.Width, p.Height)
	for i := 0; i < len(p.Pix); i++ {
		pl.Pix[i] = float64(p.Pix[i]) / 255.0
	}
	return pl
}

// Decompose separates the image to RGB channels.
func (p Pixels) Decompose() (r, g, b, a *Pixels, err error) {
	if len(p.Pix) != p.Width*p.Height*4 {
		const msg = `decompose error, widht:%d, height:%d, buf len:%d`
		err = fmt.Errorf(msg, p.Width, p.Height, len(p.Pix))
		return
	}
	r = NewPixels(p.Width, p.Height)
	g = NewPixels(p.Width, p.Height)
	b = NewPixels(p.Width, p.Height)
	a = NewPixels(p.Width, p.Height)
	for w := 0; w < p.Width; w++ {
		for h := 0; h < p.Height; h++ {
			i := w + h*p.Width
			r.Pix[i] = p.Pix[4*(w+h*p.Width)]
			g.Pix[i] = p.Pix[4*(w+h*p.Width)+1]
			b.Pix[i] = p.Pix[4*(w+h*p.Width)+2]
			a.Pix[i] = p.Pix[4*(w+h*p.Width)+3]
		}
	}
	return
}

// Compose integrates RGB channels to the image.
func Compose(r, g, b, a *Pixels) (*Pixels, error) {
	w := r.Width
	h := r.Height
	if g.Width != w || g.Height != h {
		const msg = `invalid argument error, R(%d, %d), G(%d, %d)`
		return nil, fmt.Errorf(msg, w, h, g.Width, g.Height)
	}
	if b.Width != w || b.Height != h {
		const msg = `invalid argument error, R(%d, %d), B(%d, %d)`
		return nil, fmt.Errorf(msg, w, h, b.Width, b.Height)
	}
	if a.Width != w || a.Height != h {
		const msg = `invalid argument error, R(%d, %d), A(%d, %d)`
		return nil, fmt.Errorf(msg, w, h, a.Width, a.Height)
	}
	pix := make([]uint8, 4*w*h)
	for i := 0; i < w*h; i++ {
		pix[4*i] = r.Pix[i]
		pix[4*i+1] = g.Pix[i]
		pix[4*i+2] = b.Pix[i]
		pix[4*i+3] = a.Pix[i]
	}
	return &Pixels{Width: w, Height: h, Pix: pix}, nil
}

// NewExtendPixels creates the scaled image.
func (p Pixels) NewExtendPixels(scale float64) (*Pixels, error) {
	if scale < 1.0 {
		return nil, fmt.Errorf("too small scale %d < 1.0", scale)
	}
	width := int(math.Floor(float64(p.Width)*scale + 0.5))   //round
	height := int(math.Floor(float64(p.Height)*scale + 0.5)) //round
	scaled := NewPixels(width, height)
	for w := 0; w < width; w++ {
		for h := 0; h < height; h++ {
			w0 := int(math.Floor((float64(w+1)/scale)+0.5) - 1) //round
			if w0 < 0 {
				w0 = 0
			}
			h0 := int(math.Floor((float64(h+1)/scale)+0.5) - 1) //round
			if h0 < 0 {
				h0 = 0
			}
			scaled.Pix[w+(h*width)] = p.Pix[w0+(h0*p.Width)]
		}
	}
	return scaled, nil
}
