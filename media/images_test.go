package media

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestImage creates a test image of the specified dimensions and format
func createTestImage(t *testing.T, path string, width, height int, format string) {
	t.Helper()

	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a gradient pattern for visual differentiation
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			}
			img.Set(x, y, c)
		}
	}

	// Create output file
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	// Encode in the specified format
	switch format {
	case "jpg", "jpeg":
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(f, img)
	}
	require.NoError(t, err)
}

// TestImageOrientation tests the ImageOrientation constants
func TestImageOrientation(t *testing.T) {
	assert.Equal(t, ImageOrientation(0), OrientationUnknown)
	assert.Equal(t, ImageOrientation(1), OrientationPortrait)
	assert.Equal(t, ImageOrientation(2), OrientationLandscape)
	assert.Equal(t, ImageOrientation(3), OrientationSquare)
}

// TestImageRescale tests the ImageRescale function
func TestImageRescale(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		inputWidth     int
		inputHeight    int
		inputFormat    string
		desiredWidth   int
		desiredHeight  int
		expectError    bool
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "ResizeByWidth_JPEG",
			inputWidth:     800,
			inputHeight:    600,
			inputFormat:    "jpg",
			desiredWidth:   400,
			desiredHeight:  0,
			expectError:    false,
			expectedWidth:  400,
			expectedHeight: 300,
		},
		{
			name:           "ResizeByHeight_JPEG",
			inputWidth:     800,
			inputHeight:    600,
			inputFormat:    "jpg",
			desiredWidth:   0,
			desiredHeight:  300,
			expectError:    false,
			expectedWidth:  400,
			expectedHeight: 300,
		},
		{
			name:           "ResizeBoth_PNG",
			inputWidth:     800,
			inputHeight:    600,
			inputFormat:    "png",
			desiredWidth:   200,
			desiredHeight:  200,
			expectError:    false,
			expectedWidth:  200,
			expectedHeight: 200,
		},
		{
			name:          "BothZero_Error",
			inputWidth:    800,
			inputHeight:   600,
			inputFormat:   "jpg",
			desiredWidth:  0,
			desiredHeight: 0,
			expectError:   true,
		},
		{
			name:           "LandscapeToPor trait_PNG",
			inputWidth:     1920,
			inputHeight:    1080,
			inputFormat:    "png",
			desiredWidth:   0,
			desiredHeight:  1080,
			expectError:    false,
			expectedHeight: 1080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputPath := filepath.Join(tmpDir, "input_"+tt.name+"."+tt.inputFormat)
			outputPath := filepath.Join(tmpDir, "output_"+tt.name+"."+tt.inputFormat)

			// Create test input image
			createTestImage(t, inputPath, tt.inputWidth, tt.inputHeight, tt.inputFormat)

			// Perform rescale
			err := ImageRescale(inputPath, outputPath, tt.desiredWidth, tt.desiredHeight)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify output file exists
			assert.FileExists(t, outputPath)

			// Verify output dimensions
			outFile, err := os.Open(outputPath)
			require.NoError(t, err)
			defer outFile.Close()

			config, _, err := image.DecodeConfig(outFile)
			require.NoError(t, err)

			if tt.expectedWidth > 0 {
				assert.Equal(t, tt.expectedWidth, config.Width, "width should match")
			}
			if tt.expectedHeight > 0 {
				assert.Equal(t, tt.expectedHeight, config.Height, "height should match")
			}
		})
	}
}

// TestImageRescale_InvalidInput tests error conditions
func TestImageRescale_InvalidInput(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		inputPath   string
		outputPath  string
		createInput bool
		format      string
	}{
		{
			name:        "NonExistentInputFile",
			inputPath:   filepath.Join(tmpDir, "nonexistent.jpg"),
			outputPath:  filepath.Join(tmpDir, "output.jpg"),
			createInput: false,
		},
		{
			name:        "InvalidOutputPath",
			inputPath:   filepath.Join(tmpDir, "input.jpg"),
			outputPath:  filepath.Join(tmpDir, "nonexistent/output.jpg"),
			createInput: true,
			format:      "jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createInput {
				createTestImage(t, tt.inputPath, 100, 100, tt.format)
			}

			err := ImageRescale(tt.inputPath, tt.outputPath, 50, 50)
			assert.Error(t, err)
		})
	}
}

// TestImageRescaleAutofill tests the ImageRescaleAutofill function
func TestImageRescaleAutofill(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		inputWidth    int
		inputHeight   int
		inputFormat   string
		forcedWidth   int
		desiredHeight int
		expectError   bool
	}{
		{
			name:          "Portrait_To_Square_JPEG",
			inputWidth:    600,
			inputHeight:   800,
			inputFormat:   "jpg",
			forcedWidth:   500,
			desiredHeight: 500,
			expectError:   false,
		},
		{
			name:          "Landscape_To_Square_PNG",
			inputWidth:    1920,
			inputHeight:   1080,
			inputFormat:   "png",
			forcedWidth:   800,
			desiredHeight: 800,
			expectError:   false,
		},
		{
			name:          "Square_To_Wide_JPEG",
			inputWidth:    500,
			inputHeight:   500,
			inputFormat:   "jpg",
			forcedWidth:   1000,
			desiredHeight: 500,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputPath := filepath.Join(tmpDir, "input_autofill_"+tt.name+"."+tt.inputFormat)
			outputPath := filepath.Join(tmpDir, "output_autofill_"+tt.name+"."+tt.inputFormat)

			// Create test input image
			createTestImage(t, inputPath, tt.inputWidth, tt.inputHeight, tt.inputFormat)

			// Perform autofill rescale
			err := ImageRescaleAutofill(inputPath, outputPath, tt.forcedWidth, tt.desiredHeight)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify output file exists
			assert.FileExists(t, outputPath)

			// Verify output dimensions match forced dimensions
			outFile, err := os.Open(outputPath)
			require.NoError(t, err)
			defer outFile.Close()

			config, _, err := image.DecodeConfig(outFile)
			require.NoError(t, err)

			assert.Equal(t, tt.forcedWidth, config.Width, "width should match forced width")
			assert.Equal(t, tt.desiredHeight, config.Height, "height should match desired height")
		})
	}
}

// TestImageRescaleAutofill_InvalidInput tests error conditions
func TestImageRescaleAutofill_InvalidInput(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		inputPath   string
		outputPath  string
		createInput bool
	}{
		{
			name:        "NonExistentInputFile",
			inputPath:   filepath.Join(tmpDir, "nonexistent.jpg"),
			outputPath:  filepath.Join(tmpDir, "output.jpg"),
			createInput: false,
		},
		{
			name:        "InvalidOutputPath",
			inputPath:   filepath.Join(tmpDir, "input_autofill.jpg"),
			outputPath:  filepath.Join(tmpDir, "nonexistent/output.jpg"),
			createInput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createInput {
				createTestImage(t, tt.inputPath, 100, 100, "jpg")
			}

			err := ImageRescaleAutofill(tt.inputPath, tt.outputPath, 200, 200)
			assert.Error(t, err)
		})
	}
}

// TestImageRescaleAutofill_UnsupportedFormat tests unsupported format handling
func TestImageRescaleAutofill_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()

	inputPath := filepath.Join(tmpDir, "input.jpg")
	outputPath := filepath.Join(tmpDir, "output.bmp") // Unsupported format

	createTestImage(t, inputPath, 100, 100, "jpg")

	err := ImageRescaleAutofill(inputPath, outputPath, 200, 200)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported output format")
}

// TestIsPortrait tests portrait orientation detection
func TestIsPortrait(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		width       int
		height      int
		format      string
		expectTrue  bool
		expectError bool
	}{
		{
			name:       "Portrait_600x800",
			width:      600,
			height:     800,
			format:     "jpg",
			expectTrue: true,
		},
		{
			name:       "Landscape_800x600",
			width:      800,
			height:     600,
			format:     "jpg",
			expectTrue: false,
		},
		{
			name:       "Square_500x500",
			width:      500,
			height:     500,
			format:     "png",
			expectTrue: false,
		},
		{
			name:       "Portrait_1080x1920",
			width:      1080,
			height:     1920,
			format:     "png",
			expectTrue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imagePath := filepath.Join(tmpDir, tt.name+"."+tt.format)
			createTestImage(t, imagePath, tt.width, tt.height, tt.format)

			isPortrait, err := IsPortrait(imagePath)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectTrue, isPortrait, "portrait detection should match expected")
		})
	}
}

// TestIsLandscape tests landscape orientation detection
func TestIsLandscape(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		width       int
		height      int
		format      string
		expectTrue  bool
		expectError bool
	}{
		{
			name:       "Landscape_1920x1080",
			width:      1920,
			height:     1080,
			format:     "jpg",
			expectTrue: true,
		},
		{
			name:       "Portrait_600x800",
			width:      600,
			height:     800,
			format:     "jpg",
			expectTrue: false,
		},
		{
			name:       "Square_400x400",
			width:      400,
			height:     400,
			format:     "png",
			expectTrue: false,
		},
		{
			name:       "Landscape_1600x900",
			width:      1600,
			height:     900,
			format:     "png",
			expectTrue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imagePath := filepath.Join(tmpDir, tt.name+"."+tt.format)
			createTestImage(t, imagePath, tt.width, tt.height, tt.format)

			isLandscape, err := IsLandscape(imagePath)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectTrue, isLandscape, "landscape detection should match expected")
		})
	}
}

// TestIsPortrait_InvalidInput tests error conditions
func TestIsPortrait_InvalidInput(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		imagePath string
	}{
		{
			name:      "NonExistentFile",
			imagePath: filepath.Join(tmpDir, "nonexistent.jpg"),
		},
		{
			name:      "InvalidPath",
			imagePath: filepath.Join(tmpDir, "invalid/path/image.jpg"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := IsPortrait(tt.imagePath)
			assert.Error(t, err)
		})
	}
}

// TestIsLandscape_InvalidInput tests error conditions
func TestIsLandscape_InvalidInput(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		imagePath string
	}{
		{
			name:      "NonExistentFile",
			imagePath: filepath.Join(tmpDir, "nonexistent.jpg"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := IsLandscape(tt.imagePath)
			assert.Error(t, err)
		})
	}
}

// TestDecodeImage tests the decodeImage helper function
func TestDecodeImage(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		format      string
		expectError bool
	}{
		{
			name:        "JPEG_Image",
			format:      "jpg",
			expectError: false,
		},
		{
			name:        "PNG_Image",
			format:      "png",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imagePath := filepath.Join(tmpDir, "test_decode."+tt.format)
			createTestImage(t, imagePath, 100, 100, tt.format)

			file, err := os.Open(imagePath)
			require.NoError(t, err)
			defer file.Close()

			img, err := decodeImage(file)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, img)
			assert.Equal(t, 100, img.Bounds().Dx())
			assert.Equal(t, 100, img.Bounds().Dy())
		})
	}
}

// TestDrawBackground tests the drawBackground helper function
func TestDrawBackground(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		color  color.Color
	}{
		{
			name:   "Red_100x100",
			width:  100,
			height: 100,
			color:  color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:   "Blue_50x50",
			width:  50,
			height: 50,
			color:  color.RGBA{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:   "White_200x150",
			width:  200,
			height: 150,
			color:  color.RGBA{R: 255, G: 255, B: 255, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, tt.width, tt.height))
			drawBackground(img, tt.color)

			// Verify a few random pixels have the correct color
			testPoints := [][2]int{
				{0, 0},                        // Top-left
				{tt.width - 1, 0},             // Top-right
				{0, tt.height - 1},            // Bottom-left
				{tt.width - 1, tt.height - 1}, // Bottom-right
				{tt.width / 2, tt.height / 2}, // Center
			}

			for _, pt := range testPoints {
				x, y := pt[0], pt[1]
				r, g, b, a := img.At(x, y).RGBA()
				expected_r, expected_g, expected_b, expected_a := tt.color.RGBA()

				assert.Equal(t, expected_r, r, "red component should match at (%d,%d)", x, y)
				assert.Equal(t, expected_g, g, "green component should match at (%d,%d)", x, y)
				assert.Equal(t, expected_b, b, "blue component should match at (%d,%d)", x, y)
				assert.Equal(t, expected_a, a, "alpha component should match at (%d,%d)", x, y)
			}
		})
	}
}

// TestDrawImage tests the drawImage helper function
func TestDrawImage(t *testing.T) {
	tests := []struct {
		name      string
		dstWidth  int
		dstHeight int
		srcWidth  int
		srcHeight int
		offsetX   int
		offsetY   int
	}{
		{
			name:      "Centered_50x50_in_100x100",
			dstWidth:  100,
			dstHeight: 100,
			srcWidth:  50,
			srcHeight: 50,
			offsetX:   25,
			offsetY:   25,
		},
		{
			name:      "TopLeft_30x30_in_100x100",
			dstWidth:  100,
			dstHeight: 100,
			srcWidth:  30,
			srcHeight: 30,
			offsetX:   0,
			offsetY:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := image.NewRGBA(image.Rect(0, 0, tt.dstWidth, tt.dstHeight))
			src := image.NewRGBA(image.Rect(0, 0, tt.srcWidth, tt.srcHeight))

			// Fill source with a distinct color
			srcColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
			drawBackground(src, srcColor)

			// Fill destination with a different color
			dstColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}
			drawBackground(dst, dstColor)

			// Draw source onto destination
			drawImage(dst, src, image.Point{tt.offsetX, tt.offsetY})

			// Verify source region has source color
			centerX := tt.offsetX + tt.srcWidth/2
			centerY := tt.offsetY + tt.srcHeight/2
			r, g, b, _ := dst.At(centerX, centerY).RGBA()
			assert.Equal(t, uint32(65535), r, "red should be max in source region")
			assert.Equal(t, uint32(0), g, "green should be 0 in source region")
			assert.Equal(t, uint32(0), b, "blue should be 0 in source region")

			// Verify a pixel outside source region has destination color (if possible)
			if tt.offsetX+tt.srcWidth < tt.dstWidth && tt.offsetY+tt.srcHeight < tt.dstHeight {
				outsideX := tt.offsetX + tt.srcWidth + 1
				outsideY := tt.offsetY + tt.srcHeight + 1
				if outsideX < tt.dstWidth && outsideY < tt.dstHeight {
					r, g, b, _ = dst.At(outsideX, outsideY).RGBA()
					assert.Equal(t, uint32(0), r, "red should be 0 outside source region")
					assert.Equal(t, uint32(0), g, "green should be 0 outside source region")
					assert.Equal(t, uint32(65535), b, "blue should be max outside source region")
				}
			}
		})
	}
}

// BenchmarkImageRescale benchmarks image rescaling
func BenchmarkImageRescale(b *testing.B) {
	tmpDir := b.TempDir()
	inputPath := filepath.Join(tmpDir, "input.jpg")
	outputPath := filepath.Join(tmpDir, "output.jpg")

	// Create test image
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	f, _ := os.Create(inputPath)
	jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ImageRescale(inputPath, outputPath, 640, 480)
	}
}

// BenchmarkImageRescaleAutofill benchmarks autofill rescaling
func BenchmarkImageRescaleAutofill(b *testing.B) {
	tmpDir := b.TempDir()
	inputPath := filepath.Join(tmpDir, "input.jpg")
	outputPath := filepath.Join(tmpDir, "output.jpg")

	// Create test image
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	f, _ := os.Create(inputPath)
	jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ImageRescaleAutofill(inputPath, outputPath, 800, 600)
	}
}
