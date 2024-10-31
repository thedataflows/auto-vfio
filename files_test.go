package main

import (
	"io/fs"
	"os"
	"reflect"
	"testing"
)

func TestListFiles(t *testing.T) {
	// Setup
	parentDirectory := "test"
	fileType := fs.ModeDir
	expected := []string{"test", "test/1", "test/2"}
	defer func() {
		err := os.RemoveAll(parentDirectory)
		if err != nil {
			t.Fatalf("error removing directory: %v", err)
		}
	}()
	for _, d := range expected {
		err := os.MkdirAll(d, 0755)
		if err != nil {
			t.Fatalf("error creating directory: %v", err)
		}
	}

	// Test
	actual, err := listFiles(parentDirectory, fileType)

	// Assert
	if err != nil {
		t.Fatalf("error walking directory: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected %v, but got %v", expected, actual)
	}
}
