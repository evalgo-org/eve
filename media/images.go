// Package media provides utilities for image processing and manipulation.
// It includes functions for resizing images, checking image orientation,
// and handling EXIF metadata.
//
// Features:
//   - Image rescaling with aspect ratio preservation
//   - Image rescaling with auto-fill background
//   - EXIF orientation detection
//   - Portrait/landscape orientation checking
//   - Support for JPEG and PNG formats
//   - High-quality resizing using Lanczos3 algorithm
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

// ImageOrientation represents the orientation of an image.
// Used to determine if an image is in portrait, landscape, or square format.
type ImageOrientation int

const (
	// OrientationUnknown represents an image with unknown orientation
	OrientationUnknown ImageOrientation = iota

	// OrientationPortrait represents an image in portrait orientation (height > width)
	OrientationPortrait

	// OrientationLandscape represents an image in landscape orientation (width > height)
	OrientationLandscape

	// OrientationSquare represents a square image (width = height)
	OrientationSquare
)

// ImageInfo contains metadata about an image including dimensions, orientation, and format.
// Used to store information about an image for processing decisions.
type ImageInfo struct {
	Width           int              // Image width in pixels
	Height          int              // Image height in pixels
	Orientation     ImageOrientation // Image orientation (portrait, landscape, square)
	EXIFOrientation int              // EXIF orientation tag value
	Format          string           // Image format (e.g., "jpg", "png")
}

// ImageRescale resizes an image to the desired dimensions.
// This function supports maintaining aspect ratio by setting either width or height to 0.
// If both dimensions are provided, the image may be distorted to fit exactly.
//
// Parameters:
//   - inputPath: Path to the input image file
//   - outputPath: Path to save the resized image
//   - desiredWidth: Target width in pixels (0 to maintain aspect ratio based on height)
//   - desiredHeight: Target height in pixels (0 to maintain aspect ratio based on width)
//
// Returns:
//   - error: If any step in the process fails (file operations, decoding, resizing, encoding)
//
// Supported Formats:
//   - JPEG (.jpg, .jpeg)
//   - PNG (.png)
//
// Resizing Method:
//   - Uses Lanczos3 algorithm for high-quality resizing
//   - Maintains original image format in the output
func ImageRescale(inputPath, outputPath string, desiredWidth, desiredHeight int) error {
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

	// Save the new image based on the original format
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

// decodeImage decodes an image from an io.Reader.
// Helper function to decode an image from any io.Reader source.
//
// Parameters:
//   - r: io.Reader containing the image data
//
// Returns:
//   - image.Image: The decoded image
//   - error: If decoding fails
func decodeImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

// drawBackground fills an RGBA image with a specified background color.
// Used to create a solid color background for auto-fill operations.
//
// Parameters:
//   - dst: The destination image to fill
//   - bgColor: The color to use for filling
func drawBackground(dst *image.RGBA, bgColor color.Color) {
	bounds := dst.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, y, bgColor)
		}
	}
}

// drawImage pastes one image onto another at a specified point.
// Used to position a resized image onto a background in auto-fill operations.
//
// Parameters:
//   - dst: The destination image
//   - src: The source image to paste
//   - pt: The point where the top-left corner of src should be placed in dst
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

// ImageRescaleAutofill resizes an image to fit within specified dimensions while maintaining
// aspect ratio and filling empty space with the background color.
// This function is useful for creating thumbnails or standardized images where you want
// to maintain the original aspect ratio but need a specific output size.
//
// Parameters:
//   - inputPath: Path to the input image file
//   - outputPath: Path to save the processed image
//   - forcedWidth: The exact width of the output image
//   - desiredHeight: The height to resize the image to (maintaining aspect ratio)
//
// Returns:
//   - error: If any step in the process fails
//
// Process:
//  1. Opens and decodes the input image
//  2. Uses the top-left pixel as the background color
//  3. Resizes the image to the desired height while maintaining aspect ratio
//  4. Creates a new image with the forced width and desired height
//  5. Fills the new image with the background color
//  6. Pastes the resized image centered in the new image
//  7. Saves the result in the original format
//
// Supported Formats:
//   - JPEG (.jpg, .jpeg)
//   - PNG (.png)
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

// checkOrientationWithEXIF checks an image's orientation using EXIF data.
// This function provides more accurate orientation detection than just checking dimensions,
// as it considers the EXIF orientation tag which indicates how the image should be rotated.
//
// Parameters:
//   - imagePath: Path to the image file to check
//
// Returns:
//   - *ImageInfo: Image metadata including orientation information
//   - error: If the image cannot be read or processed
//
// The function:
//  1. Opens the image file
//  2. Gets basic image dimensions
//  3. Attempts to read EXIF data
//  4. Determines orientation based on EXIF data and/or image dimensions
//  5. Returns an ImageInfo struct with the orientation information
//
// EXIF Orientation Values:
//   - 1: Normal (0°)
//   - 2: Flipped horizontally
//   - 3: Rotated 180°
//   - 4: Flipped vertically
//   - 5: Rotated 90° CCW and flipped horizontally
//   - 6: Rotated 90° CW
//   - 7: Rotated 90° CW and flipped horizontally
//   - 8: Rotated 90° CCW
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
	if _, err := file.Seek(0, 0); err != nil {
		// If seek fails, fall back to dimensions only
		if config.Width > config.Height {
			info.Orientation = OrientationLandscape
		} else if config.Height > config.Width {
			info.Orientation = OrientationPortrait
		} else {
			info.Orientation = OrientationSquare
		}
		return info, nil
	}

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

// IsPortrait checks if an image is in portrait orientation.
// This function uses EXIF data if available to determine the true orientation.
//
// Parameters:
//   - imagePath: Path to the image file to check
//
// Returns:
//   - bool: True if the image is in portrait orientation
//   - error: If the image cannot be read or processed
func IsPortrait(imagePath string) (bool, error) {
	info, err := checkOrientationWithEXIF(imagePath)
	if err != nil {
		return false, err
	}
	return info.Orientation == OrientationPortrait, nil
}

// IsLandscape checks if an image is in landscape orientation.
// This function uses EXIF data if available to determine the true orientation.
//
// Parameters:
//   - imagePath: Path to the image file to check
//
// Returns:
//   - bool: True if the image is in landscape orientation
//   - error: If the image cannot be read or processed
func IsLandscape(imagePath string) (bool, error) {
	info, err := checkOrientationWithEXIF(imagePath)
	if err != nil {
		return false, err
	}
	return info.Orientation == OrientationLandscape, nil
}
