package waifu2x

import (
	"fmt"
	"math"
)

type Pixels struct {
	Width  int
	Height int
	Pix    []uint8 // 0以下を0, 255以上を255 とする
}

func NewPixels(width, height int) *Pixels {
	return &Pixels{
		Width:  width,
		Height: height,
		Pix:    make([]uint8, width*height),
	}
}

func (p Pixels) NewImagePlane() *ImagePlane {
	pl := NewImagePlane(p.Width, p.Height)
	for i := 0; i < len(p.Pix); i++ {
		pl.Buffer[i] = float64(p.Pix[i]) / 255.0
	}
	return pl
}

func (p Pixels) Decompose() (r, g, b, a *Pixels, err error) {
	if len(p.Pix) != p.Width*p.Height*4 {
		msg := "decompose error, widht:%d, height:%d, buf len:%d"
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

func Compose(r, g, b, a *Pixels) (*Pixels, error) {
	w := r.Width
	h := r.Height
	if g.Width != w || g.Height != h {
		msg := "invalid argument error, R(%d, %d), G(%d, %d)"
		return nil, fmt.Errorf(msg, w, h, g.Width, g.Height)
	}
	if b.Width != w || b.Height != h {
		msg := "invalid argument error, R(%d, %d), B(%d, %d)"
		return nil, fmt.Errorf(msg, w, h, b.Width, b.Height)
	}
	if a.Width != w || a.Height != h {
		msg := "invalid argument error, R(%d, %d), A(%d, %d)"
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

func (p Pixels) Extrapolation(px int) *Pixels {
	ex := NewPixels(p.Width+2*px, p.Height+2*px)
	for h := 0; h < p.Height+2*px; h++ {
		for w := 0; w < p.Width+2*px; w++ {
			index := w + h*(p.Width+2*px)
			if w < px {
				// Left outer area
				if h < px {
					// Left upper area
					ex.Pix[index] = p.Pix[0]
				} else if px+p.Height <= h {
					// Left lower area
					ex.Pix[index] = p.Pix[(p.Height-1)*p.Width]
				} else {
					// Left outer area
					ex.Pix[index] = p.Pix[(h-px)*p.Width]
				}
			} else if px+p.Width <= w {
				// Right outer area
				if h < px {
					// Right upper area
					ex.Pix[index] = p.Pix[p.Width-1]
				} else if px+p.Height <= h {
					// Right lower area
					ex.Pix[index] = p.Pix[p.Width-1+(p.Height-1)*p.Width]
				} else {
					// Right outer area
					ex.Pix[index] = p.Pix[p.Width-1+(h-px)*p.Width]
				}
			} else if h < px {
				// Upper outer area
				ex.Pix[index] = p.Pix[w-px]
			} else if px+p.Height <= h {
				// Lower outer area
				ex.Pix[index] = p.Pix[w-px+(p.Height-1)*p.Width]
			} else {
				// Inner area
				ex.Pix[index] = p.Pix[w-px+(h-px)*p.Width]
			}
		}
	}
	return ex
}

func (p Pixels) NewExtendPixels(scale float64) (*Pixels, error) {
	if scale < 1.0 {
		return nil, fmt.Errorf("too small scale %d < 1.0", scale)
	}
	width := int(math.Floor(float64(p.Width)*scale + 0.5))   //Round
	height := int(math.Floor(float64(p.Height)*scale + 0.5)) //Round
	scaled := NewPixels(width, height)
	for w := 0; w < width; w++ {
		for h := 0; h < height; h++ {
			w0 := int(math.Floor((float64(w+1)/scale)+0.5) - 1) //Round
			if w0 < 0 {
				w0 = 0
			}
			h0 := int(math.Floor((float64(h+1)/scale)+0.5) - 1) //Round
			if h0 < 0 {
				h0 = 0
			}
			scaled.Pix[w+(h*width)] = p.Pix[w0+(h0*p.Width)]
		}
	}
	return scaled, nil
}
