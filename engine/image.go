package engine

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

// ReadImageFile reads the image file named by filename and returns the contents and the image format.
func ReadImageFile(r io.Reader) ([]byte, string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(b))
	return b, format, err
}
