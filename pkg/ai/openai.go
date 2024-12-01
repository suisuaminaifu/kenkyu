package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

type ExtractionResult struct {
	Title     string
	Authors   []string
	Content   string
	CreatedAt string
}

var ExtractionResultSchema = GenerateSchema[ExtractionResult]()

const EXTRACTION_RESULT_PROMPT = `
Process this research paper image and convert it to markdown format following these requirements:

1. Extract and format the following metadata if present:
   - Title
   - Author names
   - Publication/Creation date

2. Main content processing:
   - Convert all text to markdown format
   - If the paper has a two-column layout, merge it into a single-column flow
   - Preserve the logical reading order of sections
   - Keep all the equations, formulas, and mathematical expressions
   - For equations, use LaTeX format, ensure to use double dollar sign, e.g. $$E=mc^2$$
   - Be sure to keep all the lengthy content, e.g. theorems, propositions, etc.
   - Remove headers, footers, and other non-content elements
   - Maintain section headings and subheadings
   - Preserve all citations and references
   - Think logically for markdown formatting based on the content, but keep close to the original layout

3. Image handling:
   - Detect and preserve all figures, diagrams, and tables
   - For each image, create a placeholder URL
   - Generate a descriptive caption for each image
   - Insert images in their correct position in the text flow
   - Format as: ![Description](placeholder_url)

4. Output format:
   - Start with metadata (title, authors, date)
   - Follow with main content in sequential order
   - Include images with their descriptions at their original positions
   - If present, add citations/references section at the end
   - Use standard markdown formatting with LaTeX equations
   - Avoid any special formatting or complex layouts

5. Content cleanup:
   - Remove page numbers
   - Remove running headers/footers
   - Remove journal/conference formatting elements
   - Preserve only research-relevant content

Output the processed content as a single markdown file maintaining the above structure and formatting.
`

type ExtractContentFromImageArgs struct {
	OriginUrl string
	ImageUrl  string
}

func convertImageUrlToBase64Url(imageUrl string) (string, error) {
	var imageBytes []byte
	var err error

	if strings.HasPrefix(imageUrl, "http://") || strings.HasPrefix(imageUrl, "https://") {
		resp, err := http.Get(imageUrl)
		if err != nil {
			return "", fmt.Errorf("failed to fetch image from URL: %w", err)
		}
		defer resp.Body.Close()

		imageBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read image response body: %w", err)
		}
	} else {
		imageBytes, err = os.ReadFile(imageUrl)
		if err != nil {
			return "", fmt.Errorf("failed to read local image file: %w", err)
		}
	}

	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		return "", fmt.Errorf("failed to encode image as PNG: %w", err)
	}

	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/png;base64," + base64Str, nil
}

func ExtractContentFromImage(args ExtractContentFromImageArgs) (ExtractionResult, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return ExtractionResult{}, errors.New("OPENAI_API_KEY environment variable is not set")
	}

	imageUrl, err := convertImageUrlToBase64Url(args.ImageUrl)
	if err != nil {
		return ExtractionResult{}, err
	}

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("extractionResult"),
		Description: openai.F("The extraction result of the image"),
		Schema:      openai.F(ExtractionResultSchema),
		Strict:      openai.Bool(true),
	}

	client := openai.NewClient(
		option.WithAPIKey(key),
	)

	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(EXTRACTION_RESULT_PROMPT),
			openai.UserMessageParts(openai.ImagePart(imageUrl)),
		}),
		ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schemaParam),
			},
		),
		Model: openai.F(openai.ChatModelGPT4o),
	})

	if err != nil {
		return ExtractionResult{}, err
	}

	var extractionResult ExtractionResult

	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &extractionResult)
	if err != nil {
		return ExtractionResult{}, err
	}

	return extractionResult, nil
}
