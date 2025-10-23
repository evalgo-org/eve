package media

import (
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
	"github.com/rwcarlsen/goexif/exif"
)

type ImageOrientation int

const (
	OrientationUnknown ImageOrientation = iota
	OrientationPortrait
	OrientationLandscape
	OrientationSquare
)

// ImageInfo contains image metadata
type ImageInfo struct {
	Width           int
	Height          int
	Orientation     ImageOrientation
	EXIFOrientation int
	Format          string
}

func ImageRescale(inputPath, outputPath string, desiredWidth, desiredHeight int) error {
	// Desired dimensions (set to 0 to maintain aspect ratio)
	// desiredWidth := uint(0) // Target width 324
	// desiredHeight := uint(405)  // Target height (0 = maintain aspect ratio) 405

	// Open the input image file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	// Decode the image
	img, format, err := image.Decode(inputFile)
	if err != nil {
		return err
	}

	// Calculate new dimensions if maintaining aspect ratio
	var resizedImg image.Image
	if desiredWidth == 0 && desiredHeight == 0 {
		return errors.New("Either desiredWidth or desiredHeight must be greater than 0")
	} else if desiredWidth == 0 {
		// Maintain aspect ratio based on height
		ratio := float64(desiredHeight) / float64(img.Bounds().Dy())
		newWidth := uint(float64(img.Bounds().Dx()) * ratio)
		resizedImg = resize.Resize(newWidth, uint(desiredHeight), img, resize.Lanczos3)
	} else if desiredHeight == 0 {
		// Maintain aspect ratio based on width
		ratio := float64(desiredWidth) / float64(img.Bounds().Dx())
		newHeight := uint(float64(img.Bounds().Dy()) * ratio)
		resizedImg = resize.Resize(uint(desiredWidth), newHeight, img, resize.Lanczos3)
	} else {
		// Force both width and height (may distort aspect ratio)
		resizedImg = resize.Resize(uint(desiredWidth), uint(desiredHeight), img, resize.Lanczos3)
	}

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	switch format {
	case "jpg", "jpeg":
		err = jpeg.Encode(outputFile, resizedImg, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(outputFile, resizedImg)
	default:
		return errors.New("Unsupported output format: " + format)
	}

	return err
}

// decodeImage decodes an image from an io.Reader
func decodeImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

// drawBackground fills the entire image with the background color
func drawBackground(dst *image.RGBA, bgColor color.Color) {
	bounds := dst.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, y, bgColor)
		}
	}
}

// drawImage pastes src image onto dst at the specified point
func drawImage(dst *image.RGBA, src image.Image, pt image.Point) {
	srcBounds := src.Bounds()
	dstBounds := dst.Bounds()

	for y := 0; y < srcBounds.Dy(); y++ {
		for x := 0; x < srcBounds.Dx(); x++ {
			if pt.X+x >= 0 && pt.X+x < dstBounds.Dx() && pt.Y+y >= 0 && pt.Y+y < dstBounds.Dy() {
				dst.Set(pt.X+x, pt.Y+y, src.At(x, y))
			}
		}
	}
}

func ImageRescaleAutofill(inputPath, outputPath string, forcedWidth, desiredHeight int) error {
	// Open the input image file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	// Decode the image
	img, err := decodeImage(inputFile)
	if err != nil {
		return err
	}

	// Get the background color (top-left pixel in this example)
	bgColor := img.At(0, 0)

	// Resize the image to the desired height while maintaining aspect ratio
	resizedImg := resize.Resize(0, uint(desiredHeight), img, resize.Lanczos3)

	// Create a new image with the forced width and desired height
	newImg := image.NewRGBA(image.Rect(0, 0, forcedWidth, desiredHeight))

	// Fill the new image with the background color
	drawBackground(newImg, bgColor)

	// Calculate the position to paste the resized image (centered)
	xOffset := (forcedWidth - resizedImg.Bounds().Dx()) / 2

	// Paste the resized image into the new image
	drawImage(newImg, resizedImg, image.Point{xOffset, 0})

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Save the new image based on the output file extension
	ext := filepath.Ext(outputPath)
	switch ext {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(outputFile, newImg, &jpeg.Options{Quality: 90})
	case ".png":
		err = png.Encode(outputFile, newImg)
	default:
		return errors.New("Unsupported output format: " + ext)
	}

	return err
}

// Method 2: Check orientation using EXIF data (more accurate)
func checkOrientationWithEXIF(imagePath string) (*ImageInfo, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// First get basic image info
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, err
	}

	info := &ImageInfo{
		Width:  config.Width,
		Height: config.Height,
		Format: format,
	}

	// Reset file pointer for EXIF reading
	file.Seek(0, 0)

	// Try to read EXIF data
	exifData, err := exif.Decode(file)
	if err != nil {
		// No EXIF data, fall back to dimensions
		if config.Width > config.Height {
			info.Orientation = OrientationLandscape
		} else if config.Height > config.Width {
			info.Orientation = OrientationPortrait
		} else {
			info.Orientation = OrientationSquare
		}
		return info, nil
	}

	// Get orientation from EXIF
	orientationTag, err := exifData.Get(exif.Orientation)
	if err != nil {
		// No orientation in EXIF, use dimensions
		if config.Width > config.Height {
			info.Orientation = OrientationLandscape
		} else if config.Height > config.Width {
			info.Orientation = OrientationPortrait
		} else {
			info.Orientation = OrientationSquare
		}
		return info, nil
	}

	orientationValue, err := orientationTag.Int(0)
	if err != nil {
		return info, err
	}

	info.EXIFOrientation = orientationValue

	// EXIF orientation values:
	// 1 = Normal (0°)
	// 2 = Flipped horizontally
	// 3 = Rotated 180°
	// 4 = Flipped vertically
	// 5 = Rotated 90° CCW and flipped horizontally
	// 6 = Rotated 90° CW
	// 7 = Rotated 90° CW and flipped horizontally
	// 8 = Rotated 90° CCW

	switch orientationValue {
	case 1, 2, 3, 4:
		// Normal orientation or flipped (but not rotated)
		if config.Width > config.Height {
			info.Orientation = OrientationLandscape
		} else if config.Height > config.Width {
			info.Orientation = OrientationPortrait
		} else {
			info.Orientation = OrientationSquare
		}
	case 5, 6, 7, 8:
		// Rotated 90° - dimensions are swapped
		if config.Height > config.Width {
			info.Orientation = OrientationLandscape
		} else if config.Width > config.Height {
			info.Orientation = OrientationPortrait
		} else {
			info.Orientation = OrientationSquare
		}
	}

	return info, nil
}

// IsPortrait checks if image is in portrait orientation
func IsPortrait(imagePath string) (bool, error) {
	info, err := checkOrientationWithEXIF(imagePath)
	if err != nil {
		return false, err
	}
	return info.Orientation == OrientationPortrait, nil
}

// IsLandscape checks if image is in landscape orientation
func IsLandscape(imagePath string) (bool, error) {
	info, err := checkOrientationWithEXIF(imagePath)
	if err != nil {
		return false, err
	}
	return info.Orientation == OrientationLandscape, nil
}
