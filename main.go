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

	papersToReview := []ai.ReviewPaper{}

	for i := 0; i < 2; i++ {
		pdfImage, err := pdf.ConvertPdfToImage(fmt.Sprintf("test-%d.pdf", i+1))
		if err != nil {
			log.Printf("convertPdfToImage error: %v\n", err)
			return
		}

		outputDir := "output"
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Printf("failed to create output directory: %v\n", err)
			return
		}

		var title string
		log.Printf("pdfImage: %v\n", pdfImage)

		for j, imagePath := range pdfImage.ImagePaths {
			log.Printf("extracting content from image: %s\n", imagePath)
			extractionResult, err := ai.ExtractContentFromImage(ai.ExtractContentFromImageArgs{
				ImageUrl: imagePath,
			})

			if err != nil {
				log.Printf("extractContentFromImage error: %v\n", err)
				return
			}

			if j == 0 {
				title = extractionResult.Title
			}

			f, err := os.OpenFile(filepath.Join(outputDir, fmt.Sprintf("extraction_result-%d.md", i)),
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("failed to open file: %v\n", err)
				return
			}

			if _, err := f.WriteString(extractionResult.Content + "\n" + fmt.Sprintf("Page %d\n", j+1) + "\n"); err != nil {
				f.Close()
				log.Printf("failed to append to file: %v\n", err)
				return
			}
			f.Close()

			if err := os.Remove(imagePath); err != nil {
				log.Printf("failed to remove image file: %v\n", err)
				return
			}

			log.Printf("extracted page %d of %d\n", j+1, len(pdfImage.ImagePaths))
		}

		papersToReview = append(papersToReview, ai.ReviewPaper{
			PaperTitle:   title,
			PaperFileUrl: filepath.Join(outputDir, fmt.Sprintf("extraction_result-%d.md", i)),
		})
		log.Printf("added paper to review: %s, %d out of %d\n", title, i+1, 2)
	}

	reviewPaperResult, err := ai.GenerateReviewPaper(papersToReview)
	if err != nil {
		log.Printf("generateReviewPaper error: %v\n", err)
		return
	}

	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("failed to create output directory: %v\n", err)
		return
	}

	f, err := os.OpenFile(filepath.Join(outputDir, "review_paper_result.md"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open file: %v\n", err)
		return
	}

	if _, err := f.WriteString(reviewPaperResult.Content); err != nil {
		f.Close()
		log.Printf("failed to append to file: %v\n", err)
		return
	}
	f.Close()
}
