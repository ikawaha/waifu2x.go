package waifu2x

import (
	"bytes"
	"fmt"
	"image"

	"github.com/puhitaku/go-waifu2x/data"
)

func LoadModelFromAssets(fn string) (*Model, error) {
	buf, err := data.Asset(fn)
	if err != nil {
		return nil, fmt.Errorf("failed to open: %w", err)
	}
	model, err := LoadModel(bytes.NewBuffer(buf))
	if err != nil {
		return nil, fmt.Errorf("failed to load : %w", err)
	}
	return model, nil
}

func ImageToPix(img image.Image) (pix []uint8, hasAlpha bool, err error) {
	switch t := img.(type) {
	case *image.RGBA:
		pix = t.Pix
	case *image.NRGBA:
		pix = t.Pix
	case *image.YCbCr:
		var pix []uint8
		r := t.Rect
		for y := 0; y < r.Dy(); y++ {
			for x := 0; x < r.Dx(); x++ {
				r, g, b, a := t.At(x, y).RGBA()
				pix = append(pix, uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
			}
		}
	default:
		return nil, false, fmt.Errorf("unknown image format: %T", t)
	}

	for offset := 3; offset < len(pix); offset += 4 {
		if pix[offset] < 255 {
			hasAlpha = true
			return
		}
	}

	return
}

func PixToRGBA(pix []uint8, r image.Rectangle) (img *image.RGBA) {
	img = image.NewRGBA(r)
	img.Pix = pix
	img.Rect = r
	img.Stride = r.Dx() * 4
	return
}
