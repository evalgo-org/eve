// Package archive provides utilities for extracting and manipulating archive files.
// It includes secure extraction functions with path traversal protection and
// comprehensive logging for debugging and monitoring purposes.
//
// The package focuses on ZIP file extraction with built-in security measures
// to prevent directory traversal attacks (zip slip vulnerabilities) and
// proper handling of both files and directories within archives.
package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	eve "eve.evalgo.org/common"
)

// UnZip extracts all files from a ZIP archive to a specified target directory.
// This function provides secure extraction with path traversal protection,
// comprehensive logging, and proper handling of both files and directories.
//
// Security Features:
//   - Path traversal attack prevention (zip slip protection)
//   - Validates all extracted file paths remain within target directory
//   - Preserves original file permissions from the archive
//   - Creates necessary parent directories automatically
//
// Extraction Process:
//  1. Opens and validates the ZIP archive
//  2. Iterates through all entries in the archive
//  3. Validates each file path for security (prevents ../ attacks)
//  4. Creates directories or extracts files as appropriate
//  5. Preserves file permissions and directory structure
//  6. Provides detailed logging throughout the process
//
// Parameters:
//   - zipPath: Absolute or relative path to the ZIP file to extract
//   - tgtPath: Target directory where files will be extracted
//     Directory will be created if it doesn't exist
//
// Behavior:
//   - Creates target directory structure as needed
//   - Overwrites existing files with same names
//   - Preserves directory structure from the archive
//   - Logs all operations for debugging and monitoring
//   - Panics on critical errors (file I/O failures, invalid archives)
//
// Security Considerations:
//   - Prevents zip slip attacks by validating all file paths
//   - Ensures extracted files remain within the target directory
//   - Stops extraction immediately if malicious paths are detected
//   - Logs security violations for monitoring
//
// Error Handling:
//   - Panics on archive opening failures
//   - Panics on file I/O errors during extraction
//   - Silently returns on path traversal attempts (logs violation)
//   - Creates missing directories automatically
//
// Logging:
//   - Initial operation parameters (source and target paths)
//   - Each file being extracted with full path
//   - Directory creation operations
//   - Security violations (invalid file paths)
//
// Example Usage:
//
//	// Extract a ZIP file to a specific directory
//	UnZip("/path/to/archive.zip", "/path/to/extract/")
//
//	// Extract to current directory
//	UnZip("data.zip", "./extracted/")
//
// Note: This function uses panic() for error handling, which will terminate
// the program on critical failures. Consider wrapping calls in recover()
// blocks if graceful error handling is required in your application.
//
// Common Use Cases:
//   - Extracting uploaded ZIP files in web applications
//   - Processing batch data delivered as ZIP archives
//   - Extracting software packages or deployment bundles
//   - Processing backup files or data exports
func UnZip(zipPath string, tgtPath string) {
	// Log the extraction operation parameters
	eve.Logger.Info(zipPath, tgtPath)

	// Open the ZIP archive for reading
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	// Process each file/directory in the archive
	for _, f := range archive.File {
		// Construct the full file path for extraction
		filePath := filepath.Join(tgtPath, f.Name)
		eve.Logger.Info("unzipping file ", filePath)

		// Security check: Prevent path traversal attacks (zip slip)
		// Ensure the file path stays within the target directory
		if !strings.HasPrefix(filePath, filepath.Clean(tgtPath)+string(os.PathSeparator)) {
			eve.Logger.Info("invalid file path")
			return
		}

		// Handle directory entries
		if f.FileInfo().IsDir() {
			eve.Logger.Info("creating directory", filePath)
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		// Create parent directories for the file if they don't exist
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		// Create the destination file with original permissions
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		// Open the file from within the archive
		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		// Copy the file contents from archive to destination
		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		// Clean up file handles
		dstFile.Close()
		fileInArchive.Close()
	}
}
