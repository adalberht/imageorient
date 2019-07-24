// Package imageorient provides image decoding functions similar to standard library's
// image.Decode and image.DecodeConfig with the addition that they also handle the
// EXIF orientation tag (if present).
//
// See also: http://www.daveperrett.com/articles/2012/07/28/exif-orientation-handling-is-a-ghetto/
//
package imageorient

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"io/ioutil"
)

// maxBufLen is the maximum size of a buffer that should be enough to read
// the EXIF metadata. According to the EXIF specs, it is located inside the
// APP1 block that goes right after the start of image (SOI).
const maxBufLen = 1 << 20

// getOrientation returns the EXIF orientation tag from the given image
// and a new io.Reader with the same state as the original reader r.
func getOrientation(r io.Reader) (int, io.Reader) {
	buf := new(bytes.Buffer)
	tr := io.TeeReader(io.LimitReader(r, maxBufLen), buf)
	orientation := readOrientation(tr)
	return orientation, io.MultiReader(buf, r)
}

// readOrientation reads the EXIF orientation tag from the given image.
// It returns 0 if the orientation tag is not found or invalid.
func readOrientation(r io.Reader) int {
	const (
		markerSOI      = 0xffd8
		markerAPP1     = 0xffe1
		exifHeader     = 0x45786966
		byteOrderBE    = 0x4d4d
		byteOrderLE    = 0x4949
		orientationTag = 0x0112
	)

	// Check if JPEG SOI marker is present.
	var soi uint16
	if err := binary.Read(r, binary.BigEndian, &soi); err != nil {
		return 0
	}
	if soi != markerSOI {
		return 0 // Missing JPEG SOI marker.
	}

	// Find JPEG APP1 marker.
	for {
		var marker, size uint16
		if err := binary.Read(r, binary.BigEndian, &marker); err != nil {
			return 0
		}
		if err := binary.Read(r, binary.BigEndian, &size); err != nil {
			return 0
		}
		if marker>>8 != 0xff {
			return 0 // Invalid JPEG marker.
		}
		if marker == markerAPP1 {
			break
		}
		if size < 2 {
			return 0 // Invalid block size.
		}
		if _, err := io.CopyN(ioutil.Discard, r, int64(size-2)); err != nil {
			return 0
		}
	}

	// Check if EXIF header is present.
	var header uint32
	if err := binary.Read(r, binary.BigEndian, &header); err != nil {
		return 0
	}
	if header != exifHeader {
		return 0
	}
	if _, err := io.CopyN(ioutil.Discard, r, 2); err != nil {
		return 0
	}

	// Read byte order information.
	var (
		byteOrderTag uint16
		byteOrder    binary.ByteOrder
	)
	if err := binary.Read(r, binary.BigEndian, &byteOrderTag); err != nil {
		return 0
	}
	switch byteOrderTag {
	case byteOrderBE:
		byteOrder = binary.BigEndian
	case byteOrderLE:
		byteOrder = binary.LittleEndian
	default:
		return 0 // Invalid byte order flag.
	}
	if _, err := io.CopyN(ioutil.Discard, r, 2); err != nil {
		return 0
	}

	// Skip the EXIF offset.
	var offset uint32
	if err := binary.Read(r, byteOrder, &offset); err != nil {
		return 0
	}
	if offset < 8 {
		return 0 // Invalid offset value.
	}
	if _, err := io.CopyN(ioutil.Discard, r, int64(offset-8)); err != nil {
		return 0
	}

	// Read the number of tags.
	var numTags uint16
	if err := binary.Read(r, byteOrder, &numTags); err != nil {
		return 0
	}

	// Find the orientation tag.
	for i := 0; i < int(numTags); i++ {
		var tag uint16
		if err := binary.Read(r, byteOrder, &tag); err != nil {
			return 0
		}
		if tag != orientationTag {
			if _, err := io.CopyN(ioutil.Discard, r, 10); err != nil {
				return 0
			}
			continue
		}
		if _, err := io.CopyN(ioutil.Discard, r, 6); err != nil {
			return 0
		}
		var val uint16
		if err := binary.Read(r, byteOrder, &val); err != nil {
			return 0
		}
		if val < 1 || val > 8 {
			return 0 // Invalid tag value.
		}
		return int(val)
	}
	return 0 // Missing orientation tag.
}

type Decoder interface {
	Decode(r io.Reader) (image.Image, string, error)
	DecodeConfig(r io.Reader) (image.Config, string, error)
}

// Function needed to fix the given image orientation
type FixOrientationFunction func(img image.Image) (image.Image, error)

type decoder struct {
	FixOrientationFunctions map[int]FixOrientationFunction
}

// Decode decodes an image and changes its orientation
// according to the EXIF orientation tag (if present).
func (d *decoder) Decode(r io.Reader) (image.Image, string, error) {
	orientation, r := getOrientation(r)

	img, format, err := image.Decode(r)
	if err != nil {
		return img, format, err
	}
	if orientation > 1 {
		img, err = d.getFixedOrientationImage(img, orientation)
	}
	return img, format, err
}

// DecodeConfig decodes the color model and dimensions of an image
// with the respect to the EXIF orientation tag (if present).
//
// Note that after using imageorient.Decode on the same image,
// the color model of the decoded image may be different if the
// orientation-related transformation is needed.
func (d *decoder) DecodeConfig(r io.Reader) (image.Config, string, error) {
	orientation, r := getOrientation(r)

	cfg, format, err := image.DecodeConfig(r)
	if err != nil {
		return cfg, format, err
	}

	if orientation >= 5 && orientation <= 8 {
		cfg.Width, cfg.Height = cfg.Height, cfg.Width
	}

	return cfg, format, nil
}

// getFixedOrientationImage changes the image orientation based on the EXIF orientation tag value.
func (d *decoder) getFixedOrientationImage(img image.Image, orientation int) (image.Image, error) {
	filter, ok := d.FixOrientationFunctions[orientation]
	if !ok {
		return nil, errors.New(fmt.Sprintf("orientation %d not found in fixOrientationFunctions", orientation))
	}
	return filter(img)
}

func NewDecoder(fixOrientationFunctions map[int]FixOrientationFunction) Decoder {
	return &decoder{
		FixOrientationFunctions: fixOrientationFunctions,
	}
}
