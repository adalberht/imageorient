package imageorient

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"io/ioutil"
	"os"
	"testing"
)

var testFiles = []struct {
	path        string
	orientation int
}{
	{"testdata/orientation_0.jpg", 0},
	{"testdata/orientation_1.jpg", 1},
	{"testdata/orientation_2.jpg", 2},
	{"testdata/orientation_3.jpg", 3},
	{"testdata/orientation_4.jpg", 4},
	{"testdata/orientation_5.jpg", 5},
	{"testdata/orientation_6.jpg", 6},
	{"testdata/orientation_7.jpg", 7},
	{"testdata/orientation_8.jpg", 8},
}

func TestReadOrientation(t *testing.T) {
	for _, tf := range testFiles {
		f, err := os.Open(tf.path)
		if err != nil {
			t.Fatalf("os.Open(%q): %v", tf.path, err)
		}

		o := readOrientation(f)
		if o != tf.orientation {
			t.Fatalf("expected orientation=%d but got %d (%s)", tf.orientation, o, tf.path)
		}
	}
}

func TestDecodeShouldThrowErrorWhenNewDecoderIsPassingIncompleteFixOperationFunctions(t *testing.T) {
	b, err := ioutil.ReadFile(testFiles[7].path)
	if err != nil {
		t.Fatalf("%v", err)
	}

	funcs := make(map[int]FixOrientationFunction)
	d := NewDecoder(funcs)

	_, _, err = d.Decode(bytes.NewReader(b))
	if err == nil {
		t.Errorf("Wanted not nil error, got nil error")
	}
}

func TestDecodeShouldNotThrowErrorWhenNewDecoderIsPassingCompleteFixOperationFunctions(t *testing.T) {
	b, err := ioutil.ReadFile(testFiles[0].path)
	if err != nil {
		t.Fatalf("%v", err)
	}

	funcs := make(map[int]FixOrientationFunction)
	f := func(image image.Image) error {
		return nil
	}
	for i := 1; i <= 8; i++ {
		funcs[i] = f
	}
	d := NewDecoder(funcs)

	_, _, err = d.Decode(bytes.NewReader(b))
	if err != nil {
		t.Errorf("Wanted nil error, got: %v", err)
	}
}
