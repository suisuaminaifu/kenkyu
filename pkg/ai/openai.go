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
	Title     string   `json:"title" jsonschema_description:"The title of the paper"`
	Authors   []string `json:"authors" jsonschema_description:"The authors of the paper"`
	Content   string   `json:"content" jsonschema_description:"The content of the paper"`
	CreatedAt string   `json:"createdAt" jsonschema_description:"The creation date of the paper"`
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

const REVIEW_PAPER_PROMPT = `You are a research paper review generator. Your task is to analyze multiple research papers and create a comprehensive review paper that synthesizes their findings, methodologies, and conclusions. Follow these specific guidelines:

# Paper Structure and Formatting
1. Title: Create a descriptive title that encompasses the main theme of the reviewed papers
2. Abstract: Summarize the key findings and significance of the review (250-300 words)
3. Introduction:
   - Provide background context
   - State the significance of the research area
   - Define the scope of the review
   - Present clear research questions/objectives

4. Main Body:
   - Organize content thematically rather than paper-by-paper
   - Create logical sections based on key themes, methodologies, or findings
   - Use subsections for better organization
   - Include critical analysis and synthesis of findings
   - Compare and contrast different approaches
   - Highlight gaps in current research

5. Methodology Review:
   - Analyze research methods used across papers
   - Compare experimental designs
   - Evaluate data collection and analysis approaches

6. Results Synthesis:
   - Present consolidated findings
   - Use tables and figures where appropriate
   - Include statistical analyses when relevant

7. Discussion:
   - Synthesize key insights
   - Identify patterns and trends
   - Discuss contradictions or inconsistencies
   - Suggest future research directions

8. Conclusion:
   - Summarize main findings
   - State implications
   - Suggest future work

# Formatting Requirements
1. Use Markdown formatting throughout
2. Mathematical equations and formulas:
   - Enclose in $$ $$ for display equations
   - Use $ $ for inline equations
   - Example: $$E = mc^2$$

3. Images:
   - Include relevant figures from source papers
   - Format as: ![descriptive caption](placeholder_url)
   - Add detailed captions explaining significance
   - Place images strategically to support text

4. Citations:
   - Use numbered references [1], [2], etc.
   - Include in-text citations where appropriate
   - Provide full references list at end

# Content Guidelines
1. Length:
   - Expand each section fully based on available content
   - Include detailed explanations and analysis
   - Use multiple paragraphs per subtopic
   - Aim for comprehensive coverage while maintaining relevance

2. Technical Depth:
   - Maintain advanced technical level throughout
   - Explain complex concepts thoroughly
   - Include relevant technical details and specifications
   - Define specialized terms when first used

3. Analysis Quality:
   - Provide critical evaluation of methodologies
   - Identify strengths and limitations
   - Compare effectiveness of different approaches
   - Support claims with evidence from papers

4. Synthesis:
   - Draw connections between different papers
   - Identify common themes and patterns
   - Highlight contradictions and agreements
   - Suggest unified frameworks where possible

# Output Format
Generate the review paper in clean Markdown format, maintaining consistent heading levels and proper spacing. Ensure all mathematical notation, images, and citations are properly formatted according to the specified guidelines.
`

type ReviewPaper struct {
	PaperTitle   string
	PaperFileUrl string
}

type ReviewPaperResult struct {
	Title      string   `json:"title" jsonschema_description:"The title of the review paper"`
	Content    string   `json:"content" jsonschema_description:"The content of the review paper"`
	References []string `json:"references" jsonschema_description:"The references of the review paper"`
}

var ReviewPaperSchema = GenerateSchema[ReviewPaperResult]()

func GenerateReviewPaper(papers []ReviewPaper) (ReviewPaperResult, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return ReviewPaperResult{}, errors.New("OPENAI_API_KEY environment variable is not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(key),
	)

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("reviewPaper"),
		Description: openai.F("The review paper of the paper"),
		Schema:      openai.F(ReviewPaperSchema),
		Strict:      openai.Bool(true),
	}

	var papersToReviewMessages []openai.ChatCompletionMessageParamUnion

	papersToReviewMessages = append(papersToReviewMessages, openai.UserMessage(REVIEW_PAPER_PROMPT))

	for _, paper := range papers {
		content, err := os.ReadFile(paper.PaperFileUrl)
		if err != nil {
			return ReviewPaperResult{}, fmt.Errorf("failed to read paper file %s: %w", paper.PaperFileUrl, err)
		}

		paperMessage := fmt.Sprintf("Paper Title: %s\n\nContent:\n%s", paper.PaperTitle, string(content))
		papersToReviewMessages = append(papersToReviewMessages, openai.UserMessage(paperMessage))
	}

	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: openai.F(papersToReviewMessages),
		ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schemaParam),
			},
		),
		Model:               openai.F(openai.ChatModelGPT4o),
		MaxCompletionTokens: openai.Int(4096),
	})

	if err != nil {
		return ReviewPaperResult{}, err
	}

	var reviewPaperResult ReviewPaperResult

	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &reviewPaperResult)
	if err != nil {
		return ReviewPaperResult{}, err
	}

	return reviewPaperResult, nil
}
