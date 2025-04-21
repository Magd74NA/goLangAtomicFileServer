package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	totalRequests = 1000 // Total uploads to test
	concurrency   = 100  // Simultaneous workers
)

func main() {

	url := "http://localhost:8080/file"

	var success, failures atomic.Int64
	var wg sync.WaitGroup

	// Worker pool
	jobs := make(chan int, totalRequests)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for range jobs {
				if err := uploadFile(url); err != nil {
					failures.Add(1)
					fmt.Printf("Error")
				} else {
					success.Add(1)
					fmt.Printf("success")
				}
			}
		}()
	}
	for i := 0; i < totalRequests; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	fmt.Printf("\n success: %d | Failures: %d \n", success.Load(), failures.Load())
}

func uploadFile(url string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create test file
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		return err
	}
	if _, err = part.Write(bytes.Repeat([]byte("test"), 1000)); err != nil {
		return err
	}
	writer.Close()

	// Send request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Verify uniqueness in response
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if !strings.Contains(string(resBody), "uploaded successfully") {
		return fmt.Errorf("unexpected response: %s", resBody)
	}

	return nil
}
