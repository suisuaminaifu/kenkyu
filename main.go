package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/suisuaminaifu/kenkyu/pkg/ai"
	"github.com/suisuaminaifu/kenkyu/pkg/pdf"
)

func main() {
	fmt.Println("Hello, World from Kenkyu!")

	pdfImage, err := pdf.ConvertPdfToImage("test.pdf")
	if err != nil {
		log.Printf("convertPdfToImage error: %v\n", err)
		return
	}

	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("failed to create output directory: %v\n", err)
		return
	}

	for i, imagePath := range pdfImage.ImagePaths {
		extractionResult, err := ai.ExtractContentFromImage(ai.ExtractContentFromImageArgs{
			ImageUrl: imagePath,
		})

		if err != nil {
			log.Printf("extractContentFromImage error: %v\n", err)
			return
		}

		f, err := os.OpenFile(filepath.Join(outputDir, "extraction_result.md"),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("failed to open file: %v\n", err)
			return
		}

		if _, err := f.WriteString(extractionResult.Content + "\n" + fmt.Sprintf("Page %d\n", i+1) + "\n"); err != nil {
			f.Close()
			log.Printf("failed to append to file: %v\n", err)
			return
		}
		f.Close()

		if err := os.Remove(imagePath); err != nil {
			log.Printf("failed to remove image file: %v\n", err)
			return
		}

		log.Printf("extracted page %d of %d\n", i+1, len(pdfImage.ImagePaths))
	}

}
