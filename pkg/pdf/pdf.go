package pdf

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"strings"
)

type PdfImage struct {
	PdfPath    string
	ImagePaths []string
}

func ConvertPdfToImage(pdfPath string) (PdfImage, error) {
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
		log.Printf("pdftoppm error: %v\n", stderr.String())
		return PdfImage{}, err
	}

	imagePaths := []string{}
	imagePathLines := strings.Split(stderr.String(), "\n")
	for _, imagePathLine := range imagePathLines {
		if imagePathLine == "" {
			continue
		}

		log.Printf("imagePathLine: %s\n", imagePathLine)
		if strings.HasPrefix(imagePathLine, "Syntax Error") {
			continue
		}

		imagePaths = append(imagePaths, strings.Split(imagePathLine, " ")[2])
	}

	return PdfImage{
		PdfPath:    pdfPath,
		ImagePaths: imagePaths,
	}, nil
}
