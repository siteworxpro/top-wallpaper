package resize

import (
	"bytes"
	"github.com/nfnt/resize"
	i "image"
	"image/jpeg"
	_ "image/png"
)

func Shrink(image string, maxSize uint, quality int) (string, error) {
	img, _, err := i.Decode(bytes.NewReader([]byte(image)))
	if err != nil {
		return "", err
	}

	resizedImg := resize.Thumbnail(maxSize, maxSize, img, resize.Lanczos3)
	var buffer bytes.Buffer
	err = jpeg.Encode(&buffer, resizedImg, &jpeg.Options{Quality: quality})
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}
