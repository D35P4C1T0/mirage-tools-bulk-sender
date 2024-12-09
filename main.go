package main

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// isImage checks if the file is an image based on its extension
func isImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png"
}

// sendImage sends an image to the endpoint and retries if necessary
func sendImage(client *resty.Client, folderPath, endpointURL, filePath string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Failed to open file %s: %v\n", filePath, err)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("Failed to close file %s: %v\n", filePath, err)
		}
	}(file)

	// Retry logic
	var lastResponse *resty.Response
	var retryCount int
	for retryCount < 4 { // Maximum 3 retries
		// Send the image as a multipart/form-data request
		lastResponse, err = client.R().
			SetFile("file", filePath).
			SetHeader("Content-Type", "multipart/form-data").
			SetHeader("Accept", "application/json").
			Post(endpointURL)

		if err != nil {
			fmt.Printf("Failed to send image %s: %v\n", filePath, err)
			return
		}

		// Check if the response status code is 200 or 201
		if lastResponse.StatusCode() == 200 || lastResponse.StatusCode() == 201 {
			fmt.Printf("Successfully sent: %s\n", filePath)
			return
		}

		// If the status code is not 200/201, wait for a brief period and retry
		fmt.Printf("Failed to send image %s, received status code %d. Retrying...\n", filePath, lastResponse.StatusCode())
		retryCount++
		time.Sleep(10 * time.Second) // Wait for 2 seconds before retrying
	}

	// If we exhausted retries, report failure
	if retryCount >= 4 {
		fmt.Printf("Failed to send image %s after multiple retries\n", filePath)
	}
}

// sendImagesFromFolder sends all images in the specified folder to the endpoint
func sendImagesFromFolder(folderPath, endpointURL string) {
	client := resty.New()
	var wg sync.WaitGroup

	// Read all files in the folder
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Failed to read file %s: %v\n", path, err)
			return nil
		}

		// Only process image files
		if info.IsDir() || !isImage(path) {
			return nil
		}

		// Start a goroutine for sending the image
		wg.Add(1)
		go sendImage(client, folderPath, endpointURL, path, &wg)

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the directory: %v\n", err)
	}

	// Wait for all goroutines to finish
	wg.Wait()
}

func main() {
	// Get the command line arguments
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <folder_path> <endpoint_url>")
		return
	}

	folderPath := os.Args[1]
	endpointURL := os.Args[2]

	// Send the images from the folder to the endpoint
	sendImagesFromFolder(folderPath, endpointURL)
}
