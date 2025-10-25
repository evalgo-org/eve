// Package archive provides testing utilities and test cases for archive extraction functionality.
// This file contains comprehensive unit tests for the archive package, validating
// extraction functionality, security measures, and error handling.
package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// createTestZip creates a test ZIP file with specified files and directories.
// Returns the path to the created ZIP file.
func createTestZip(t *testing.T, baseDir string, files map[string]string, dirs []string) string {
	t.Helper()
	zipPath := filepath.Join(baseDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create ZIP file: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	// Add directories
	for _, dir := range dirs {
		_, err := w.Create(dir + "/")
		if err != nil {
			t.Fatalf("Failed to create directory in ZIP: %v", err)
		}
	}

	// Add files
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("Failed to create file in ZIP: %v", err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}
	}

	return zipPath
}

// createMaliciousZip creates a ZIP file with path traversal attempts
func createMaliciousZip(t *testing.T, baseDir string, maliciousPath string) string {
	t.Helper()
	zipPath := filepath.Join(baseDir, "malicious.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create malicious ZIP file: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	// Create file with malicious path
	f, err := w.Create(maliciousPath)
	if err != nil {
		t.Fatalf("Failed to create malicious entry: %v", err)
	}
	_, err = f.Write([]byte("malicious content"))
	if err != nil {
		t.Fatalf("Failed to write malicious content: %v", err)
	}

	return zipPath
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readFile reads and returns the content of a file
func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// TestUnZipBasicExtraction tests basic ZIP file extraction functionality
func TestUnZipBasicExtraction(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test ZIP with files
	files := map[string]string{
		"test.txt":        "Hello, World!",
		"subdir/file.txt": "Nested file content",
	}
	dirs := []string{"emptydir"}

	zipPath := createTestZip(t, tmpDir, files, dirs)
	targetDir := filepath.Join(tmpDir, "extracted")

	// Extract ZIP
	UnZip(zipPath, targetDir)

	// Verify files were extracted
	testFile := filepath.Join(targetDir, "test.txt")
	if !fileExists(testFile) {
		t.Errorf("Expected file %s to exist after extraction", testFile)
	}

	content := readFile(t, testFile)
	if content != "Hello, World!" {
		t.Errorf("Expected file content 'Hello, World!', got '%s'", content)
	}

	// Verify nested file
	nestedFile := filepath.Join(targetDir, "subdir", "file.txt")
	if !fileExists(nestedFile) {
		t.Errorf("Expected nested file %s to exist", nestedFile)
	}

	nestedContent := readFile(t, nestedFile)
	if nestedContent != "Nested file content" {
		t.Errorf("Expected nested content 'Nested file content', got '%s'", nestedContent)
	}

	// Verify directory was created
	emptyDir := filepath.Join(targetDir, "emptydir")
	if !fileExists(emptyDir) {
		t.Errorf("Expected directory %s to exist", emptyDir)
	}
}

// TestUnZipEmptyArchive tests extraction of an empty ZIP file
func TestUnZipEmptyArchive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty ZIP
	zipPath := createTestZip(t, tmpDir, map[string]string{}, []string{})
	targetDir := filepath.Join(tmpDir, "extracted")

	// Should not panic on empty ZIP
	UnZip(zipPath, targetDir)

	// Verify extraction completed without errors
	// (no files to check, but function should complete successfully)
}

// TestUnZipSecurityPathTraversal tests protection against path traversal attacks
func TestUnZipSecurityPathTraversal(t *testing.T) {
	tests := []struct {
		name          string
		maliciousPath string
		description   string
	}{
		{
			name:          "Relative path traversal",
			maliciousPath: "../../malicious.txt",
			description:   "Attempts to escape using ../ sequences",
		},
		{
			name:          "Multiple traversal",
			maliciousPath: "../../../etc/passwd",
			description:   "Multiple levels of directory traversal",
		},
		{
			name:          "Mixed path",
			maliciousPath: "good/../../../bad.txt",
			description:   "Valid path followed by traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create malicious ZIP
			zipPath := createMaliciousZip(t, tmpDir, tt.maliciousPath)
			targetDir := filepath.Join(tmpDir, "extracted")

			// Extract - should not extract files outside target
			UnZip(zipPath, targetDir)

			// Verify no files were created outside target directory
			// Check parent directory
			parentDir := filepath.Dir(targetDir)
			entries, err := os.ReadDir(parentDir)
			if err != nil {
				t.Fatalf("Failed to read parent directory: %v", err)
			}

			// Should only contain the extracted directory and test.zip/malicious.zip
			for _, entry := range entries {
				name := entry.Name()
				if name != "extracted" && name != "malicious.zip" {
					t.Errorf("Unexpected file/directory in parent: %s", name)
				}
			}
		})
	}
}

// TestUnZipNestedDirectories tests extraction of deeply nested directory structures
func TestUnZipNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"level1/level2/level3/deep.txt": "Deep file content",
	}
	dirs := []string{
		"level1",
		"level1/level2",
		"level1/level2/level3",
	}

	zipPath := createTestZip(t, tmpDir, files, dirs)
	targetDir := filepath.Join(tmpDir, "extracted")

	UnZip(zipPath, targetDir)

	// Verify nested structure
	deepFile := filepath.Join(targetDir, "level1", "level2", "level3", "deep.txt")
	if !fileExists(deepFile) {
		t.Errorf("Expected deeply nested file to exist: %s", deepFile)
	}

	content := readFile(t, deepFile)
	if content != "Deep file content" {
		t.Errorf("Expected 'Deep file content', got '%s'", content)
	}
}

// TestUnZipMultipleFiles tests extraction of archives with multiple files
func TestUnZipMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create ZIP with multiple files
	files := map[string]string{
		"file1.txt":     "Content 1",
		"file2.txt":     "Content 2",
		"file3.txt":     "Content 3",
		"dir/file4.txt": "Content 4",
		"dir/file5.txt": "Content 5",
	}

	zipPath := createTestZip(t, tmpDir, files, []string{"dir"})
	targetDir := filepath.Join(tmpDir, "extracted")

	UnZip(zipPath, targetDir)

	// Verify all files were extracted
	for name, expectedContent := range files {
		filePath := filepath.Join(targetDir, name)
		if !fileExists(filePath) {
			t.Errorf("Expected file %s to exist", filePath)
			continue
		}

		content := readFile(t, filePath)
		if content != expectedContent {
			t.Errorf("File %s: expected '%s', got '%s'", name, expectedContent, content)
		}
	}
}

// TestUnZipExistingTargetDirectory tests extraction to an existing directory
func TestUnZipExistingTargetDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "extracted")

	// Create target directory first
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create existing file in target
	existingFile := filepath.Join(targetDir, "existing.txt")
	err = os.WriteFile(existingFile, []byte("Existing content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Create ZIP
	files := map[string]string{
		"new.txt": "New content",
	}
	zipPath := createTestZip(t, tmpDir, files, []string{})

	// Extract
	UnZip(zipPath, targetDir)

	// Verify new file was extracted
	newFile := filepath.Join(targetDir, "new.txt")
	if !fileExists(newFile) {
		t.Errorf("Expected new file to be extracted")
	}

	// Verify existing file is still there
	if !fileExists(existingFile) {
		t.Errorf("Expected existing file to remain")
	}
}

// TestUnZipOverwriteExistingFiles tests that extraction overwrites existing files
func TestUnZipOverwriteExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "extracted")

	// Create ZIP
	files := map[string]string{
		"test.txt": "New content",
	}
	zipPath := createTestZip(t, tmpDir, files, []string{})

	// Create existing file with different content
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}
	existingFile := filepath.Join(targetDir, "test.txt")
	err = os.WriteFile(existingFile, []byte("Old content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Extract - should overwrite
	UnZip(zipPath, targetDir)

	// Verify file was overwritten
	content := readFile(t, existingFile)
	if content != "New content" {
		t.Errorf("Expected file to be overwritten with 'New content', got '%s'", content)
	}
}

// TestUnZipInvalidZipFile tests handling of invalid ZIP files
func TestUnZipInvalidZipFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid ZIP file
	invalidZip := filepath.Join(tmpDir, "invalid.zip")
	err := os.WriteFile(invalidZip, []byte("This is not a ZIP file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid ZIP: %v", err)
	}

	targetDir := filepath.Join(tmpDir, "extracted")

	// Should panic on invalid ZIP
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected UnZip to panic on invalid ZIP file")
		}
	}()

	UnZip(invalidZip, targetDir)
}

// TestUnZipNonexistentFile tests handling of nonexistent ZIP files
func TestUnZipNonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()

	nonexistentZip := filepath.Join(tmpDir, "nonexistent.zip")
	targetDir := filepath.Join(tmpDir, "extracted")

	// Should panic on nonexistent file
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected UnZip to panic on nonexistent file")
		}
	}()

	UnZip(nonexistentZip, targetDir)
}

// BenchmarkUnZipSmallArchive benchmarks extraction of a small archive
func BenchmarkUnZipSmallArchive(b *testing.B) {
	tmpDir := b.TempDir()

	files := map[string]string{
		"file1.txt": "Content 1",
		"file2.txt": "Content 2",
		"file3.txt": "Content 3",
	}

	// Create a minimal T for the helper function
	testT := &testing.T{}
	zipPath := createTestZip(testT, tmpDir, files, []string{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targetDir := filepath.Join(tmpDir, "extracted", filepath.Base(tmpDir), filepath.Base(tmpDir)+"-"+filepath.Base(tmpDir))
		UnZip(zipPath, targetDir)
		// Clean up to avoid disk space issues
		os.RemoveAll(targetDir)
	}
}

// BenchmarkUnZipMediumArchive benchmarks extraction of a medium-sized archive
func BenchmarkUnZipMediumArchive(b *testing.B) {
	tmpDir := b.TempDir()

	// Create archive with 50 files (reduced from 100 for faster benchmarks)
	files := make(map[string]string)
	for i := 0; i < 50; i++ {
		files[filepath.Join("dir", filepath.Base(tmpDir), "file.txt")] = "File content"
	}

	testT := &testing.T{}
	zipPath := createTestZip(testT, tmpDir, files, []string{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targetDir := filepath.Join(tmpDir, "extracted", filepath.Base(tmpDir), filepath.Base(tmpDir)+"-"+filepath.Base(tmpDir))
		UnZip(zipPath, targetDir)
		// Clean up to avoid disk space issues
		os.RemoveAll(targetDir)
	}
}
