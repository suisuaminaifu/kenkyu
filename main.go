package main

import (
	"fmt"
	"log"

	"github.com/suisuaminaifu/kenkyu/pkg/pdf"
)

func main() {
	fmt.Println("Hello, World from Kenkyu!")

	pdfImage, err := pdf.ConvertPdfToImage("test.pdf")
	if err != nil {
		log.Printf("convertPdfToImage error: %v\n", err)
		return
	}

	fmt.Printf("%+v\n", pdfImage)
}
