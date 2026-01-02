package ai

import (
	"bytes"
	"context"
	"fmt"
	
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

type ContentGenerator struct {
	client *genai.Client
	model  *genai.GenerativeModel
	md     goldmark.Markdown
}

func NewContentGenerator(apiKey string) (*ContentGenerator, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel("gemini-2.5-flash")

	// Configure Markdown parser with syntax highlighting
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"), // Dark theme
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	return &ContentGenerator{
		client: client,
		model:  model,
		md:     md,
	}, nil
}

func (c *ContentGenerator) GenerateArticle(ctx context.Context) (string, error) {
	prompt := `
	Write a crisp, concise, and highly technical system design article about a random, interesting topic in software engineering (e.g., distributed rate limiting, consistent hashing, LSM trees, raft consensus, bloom filters).
	
	OUTPUT FORMAT:
	- **Markdown** (CommonMark/GFM).
	- Do NOT wrap the entire output in a single code block.
	
	GUIDELINES:
	1. **Structure**: Use # for Title, ## for sections.
	2. **Code**: Use triple backticks (e.g. ` + "`" + `go ... ` + "`" + `) for code blocks. Ensure code is practical and idiomatic Go.
	3. **Tone**: Educational, professional, no fluff.
	
	REQUIRED SECTIONS:
	1. Title
	2. Introduction (The "Why")
	3. Core Concepts (The "What")
	4. Go Implementation Patterns (The "How" - Heavy on code)
	5. Trade-offs and Challenges
	6. Conclusion
	`

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	// Extract text from the response
	var rawMarkdown string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			rawMarkdown += string(txt)
		}
	}

	// Convert Markdown to HTML with Highlighting
	var buf bytes.Buffer
	if err := c.md.Convert([]byte(rawMarkdown), &buf); err != nil {
		return "", fmt.Errorf("markdown conversion failed: %w", err)
	}

	// Wrap in a styled HTML template
	// Note: We don't need manual code styles anymore, Chroma handles it inline/classes.
	finalHTML := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
<style>
	body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333; max-width: 800px; margin: 0 auto; padding: 20px; }
	h1 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
	h2 { color: #2980b9; margin-top: 30px; border-bottom: 1px solid #eee; padding-bottom: 5px; }
	p { margin-bottom: 15px; }
	ul, ol { margin-bottom: 20px; padding-left: 25px; }
	li { margin-bottom: 5px; }
	
	/* Code Block Container Styling */
	pre { 
		padding: 15px; 
		border-radius: 5px; 
		overflow-x: auto; 
		font-family: 'Menlo', 'Monaco', 'Courier New', monospace; 
		font-size: 14px; 
		border: 1px solid #ddd; 
		/* Background is handled by Chroma (dracula theme usually has dark bg) */
	}
	code { font-family: 'Menlo', 'Monaco', 'Courier New', monospace; }
</style>
</head>
<body>
	%s
</body>
</html>`, buf.String())

	return finalHTML, nil
}

func (c *ContentGenerator) Close() {
	c.client.Close()
}
