package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

func extractTextFromPdf(pdfPath string) (string, error) {
	args := []string{
		"-layout", // Maintain (as best as possible) the original physical layout of the text.
		pdfPath,
		"-", // Send the output to stdout.
	}

	cmd := exec.CommandContext(context.TODO(), "pdftotext", args...)

	var buf bytes.Buffer
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func main() {
	fmt.Println("Hello, World from Kenkyu!")

	result, err := extractTextFromPdf("test.pdf")
	if err != nil {
		fmt.Println("Error extracting text from PDF:", err)
		return
	}

	fmt.Println(result)
}
