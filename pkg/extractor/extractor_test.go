package extractor

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestExtract(t *testing.T) {
	// Create a temporary directory to use as the output directory
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create some files to use as input
	file1 := filepath.Join(tempDir, "file1")
	file2 := filepath.Join(tempDir, "file2")
	if err := ioutil.WriteFile(file1, []byte("file1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(file2, []byte("file2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Define a mock function for the option type
	mockOption := func(c *config) error {
		c.files = []string{file1, file2}
		return nil
	}

	// Call the Extract function with the mock option
	err = Extract(mockOption)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify that the files are copied correctly
	expectedFile1 := filepath.Join(tempDir, "file1")
	expectedFile2 := filepath.Join(tempDir, "file2")
	if _, err := os.Stat(expectedFile1); err != nil {
		t.Errorf("File not copied: %v", err)
	}
	if _, err := os.Stat(expectedFile2); err != nil {
		t.Errorf("File not copied: %v", err)
	}
}
