package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Embed everything inside the 'qaac' folder
//
//go:embed qaac/qaac64.exe qaac/QTfiles64/* qaac/libFLAC.dll qaac/libFLAC++.dll qaac/libsoxr64.dll qaac/sndfile.dll
var embeddedFiles embed.FS

// Extract a file from the embedded FS to a specified directory
func extractFile(filename, destDir string) error {
	data, err := embeddedFiles.ReadFile(filename)
	if err != nil {
		return err
	}

	destPath := filepath.Join(destDir, filepath.Base(filename))
	err = os.WriteFile(destPath, data, fs.FileMode(0755))
	if err != nil {
		return err
	}
	return nil
}

// Extract all necessary files for qaac to run
func extractAllFiles(tempDir string) error {
	// Create the QTfiles64 directory inside the temp directory
	qtFilesDir := filepath.Join(tempDir, "QTfiles64")
	err := os.MkdirAll(qtFilesDir, os.ModePerm) // Ensure the directory is created
	if err != nil {
		return fmt.Errorf("failed to create QTfiles64 directory: %v", err)
	}

	// Extract qaac64 executable
	err = extractFile("qaac/qaac64.exe", tempDir)
	if err != nil {
		return err
	}

	// Extract DLLs
	dlls := []string{"qaac/libFLAC.dll", "qaac/libFLAC++.dll", "qaac/libsoxr64.dll", "qaac/sndfile.dll"}
	for _, dll := range dlls {
		err = extractFile(dll, tempDir)
		if err != nil {
			return err
		}
	}

	// Extract all QTfiles64 DLLs
	err = fs.WalkDir(embeddedFiles, "qaac/QTfiles64", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return extractFile(path, qtFilesDir)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Run qaac64 to convert the file to ALAC
func runQaac(inputFile, outputFile, tempDir string) error {
	qaacPath := filepath.Join(tempDir, "qaac64.exe")
	cmd := exec.Command(qaacPath, "-A", inputFile, "-o", outputFile)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run qaac: %v", err)
	}

	return nil
}

// Process a single file and rename it
func processFile(inputFile, tempDir string) error {
	// Get the base name of the input file
	baseName := filepath.Base(inputFile)
	ext := filepath.Ext(baseName)
	originalName := baseName[:len(baseName)-len(ext)]

	// Generate the new output filename with the [Everything2ALAC] prefix
	outputFile := filepath.Join(filepath.Dir(inputFile), fmt.Sprintf("[Everything2ALAC]%s.alac.m4a", originalName))

	// Run the qaac conversion
	err := runQaac(inputFile, outputFile, tempDir)
	if err != nil {
		return fmt.Errorf("failed to convert file %s: %v", inputFile, err)
	}

	fmt.Printf("Successfully converted: %s -> %s\n", inputFile, outputFile)
	return nil
}

func main() {
	// Step 1: Check if there are files or directories passed via drag and drop
	if len(os.Args) < 2 {
		fmt.Println("Please drag and drop files or a directory onto the executable.")
		return
	}

	// Extract the input path(s) from os.Args
	inputPaths := os.Args[1:] // os.Args[0] is the path to the executable

	tempDir := os.TempDir() // Use system temp directory

	// Step 2: Extract necessary files for qaac to run
	err := extractAllFiles(tempDir)
	if err != nil {
		fmt.Println("Error extracting files:", err)
		return
	}

	// Step 3: Process each input path (file or directory)
	for _, inputPath := range inputPaths {
		// Check if input is a directory
		fileInfo, err := os.Stat(inputPath)
		if err != nil {
			fmt.Printf("Error accessing input path: %s\n", inputPath)
			continue
		}

		if fileInfo.IsDir() {
			// Batch process all files in the directory
			err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && (strings.HasSuffix(info.Name(), ".wav") || strings.HasSuffix(info.Name(), ".flac")) {
					// Process supported files (e.g., .wav and .flac)
					return processFile(path, tempDir)
				}
				return nil
			})
			if err != nil {
				fmt.Printf("Error processing directory: %s\n", inputPath)
				continue
			}
		} else {
			// Single file process
			err = processFile(inputPath, tempDir)
			if err != nil {
				fmt.Printf("Error processing file: %s\n", inputPath)
				continue
			}
		}
	}

	fmt.Println("All conversions completed successfully!")
}
