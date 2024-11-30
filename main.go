package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type PdfImage struct {
	PdfPath    string
	ImagePaths []string
}

func convertPdfToImage(pdfPath string) (PdfImage, error) {
	imagePath := "./tmp/tmpPdfImage"
	args := []string{
		"-png",
		"-progress",
		pdfPath,
		imagePath,
	}

	cmd := exec.CommandContext(context.TODO(), "pdftoppm", args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return PdfImage{}, err
	}

	imagePaths := []string{}
	imagePathLines := strings.Split(stderr.String(), "\n")
	for _, imagePathLine := range imagePathLines {
		if imagePathLine == "" {
			continue
		}

		imagePaths = append(imagePaths, strings.Split(imagePathLine, " ")[2])
	}

	return PdfImage{
		PdfPath:    pdfPath,
		ImagePaths: imagePaths,
	}, nil
}

func deleteImage(imagePath string) error {
	return os.Remove(imagePath)
}

func main() {
	fmt.Println("Hello, World from Kenkyu!")

	pdfImage, err := convertPdfToImage("test.pdf")
	if err != nil {
		log.Printf("convertPdfToImage error: %v\n", err)
		return
	}

	fmt.Printf("%+v\n", pdfImage)
}
