package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

func main() {
	url := "http://localhost:8080/file"

	for i := 0; i < 100; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Create test file content
		part, err := writer.CreateFormFile("file", "test.txt")
		if err != nil {
			fmt.Printf("Error creating form file: %v\n", err)
			continue
		}
		_, err = part.Write(bytes.Repeat([]byte("test"), 1000))
		if err != nil {
			fmt.Printf("Error writing to part: %v\n", err)
			continue
		}
		writer.Close()

		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			continue
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Error sending request: %v\n", err)
			continue
		}
		defer res.Body.Close()

		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			continue
		}

		fmt.Printf("Status: %s | Response: %s\n", res.Status, resBody)
	}
}
