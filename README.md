# imageorient

[![GoDoc](https://godoc.org/github.com/disintegration/imageorient?status.svg)](https://godoc.org/github.com/disintegration/imageorient)

Package `imageorient` provides image decoding functions similar to standard library's
`image.Decode` and `image.DecodeConfig` with an option to provide custom functions to fix the EXIF orientation (if present).


License: MIT.

See also: http://www.daveperrett.com/articles/2012/07/28/exif-orientation-handling-is-a-ghetto/

## Install / Update

    go get -u github.com/adalberht/imageorient

## Documentation

http://godoc.org/github.com/disintegration/imageorient

## Usage example

```go
package main

import (
    "image"
	"image/jpeg"
	"log"
	"os"
    "github.com/adalberht/imageorient"
    
	"github.com/anthonynsimon/bild/transform"
)

func main() {
	// Create new Decoder
	funcs := make(map[int]imageorient.FixOrientationFunction)
	// TODO: update docs with working example
	funcs[2] = func(img image.Image) error {
		return nil
	}
	
	d := imageorient.NewDecoder(funcs)
	
	// Open the test image. This particular image have the EXIF
	// orientation tag set to 3 (rotated by 180 deg).
	f, err := os.Open("testdata/orientation_3.jpg")
	if err != nil {
		log.Fatalf("os.Open failed: %v", err)
	}

	// Decode the test image using the imageorient.Decode function
	// to handle the image orientation correctly.
	img, _, err := d.Decode(f)
	if err != nil {
		log.Fatalf("imageorient.Decode failed: %v", err)
	}

	// Save the decoded image to a new file. If we used image.Decode
	// instead of imageorient.Decode on the previous step, the saved
	// image would appear rotated.
	f, err = os.Create("testdata/example_output.jpg")
	if err != nil {
		log.Fatalf("os.Create failed: %v", err)
	}
	err = jpeg.Encode(f, img, nil)
	if err != nil {
		log.Fatalf("jpeg.Encode failed: %v", err)
	}
}
```