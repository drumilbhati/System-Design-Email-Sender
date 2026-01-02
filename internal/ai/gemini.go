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
	You are a Senior Mentor and Technical Lead writing a daily educational newsletter for **college students and junior engineers**.
	
	Your goal is to explain complex software engineering concepts in a way that is **accessible, encouraging, and easy to understand**, while still being technically accurate. Avoid overly dense jargon; if you use a complex term, explain it simply first.
	
	Your task is to generate an article on a **randomly selected topic** from one of the following categories. Pick ONE category and one specific topic.

	### CATEGORY 1: Real-World System Breakdowns (Case Studies)
	Explain how big tech companies solve problems, but focus on the "Aha!" moments.
	- Examples: "How Discord handles so many messages", "Why Netflix doesn't crash", "How Instagram generates IDs".
	- Focus on: The simple logic behind the massive scale. Use analogies.

	### CATEGORY 2: High-Level Design (HLD)
	Design a system component, but keep it grounded.
	- Examples: "Designing a simple Job Scheduler", "How Google Docs lets two people type at once", "Building a 'Nearby Friends' feature".
	- Focus on: The basic building blocks (Database, Cache, Load Balancer) and how they talk to each other.

	### CATEGORY 3: Low-Level Design (LLD) & Internals
	Dive into code, but make it readable.
	- Examples: "How a Thread Pool actually works", "Writing your own HashMap", "Understanding Go Context with examples".
	- Focus on: Clear code examples (Go or Java) and explaining *why* we write it this way.

	### GUIDELINES:
	1. **Tone**: **Friendly, Mentorship-style**. Imagine explaining this to a smart intern. Use simple analogies (e.g., "Think of a Load Balancer like a receptionist...").
	2. **Structure**:
		- **Title**: Catchy and clear.
		- **The Problem**: Why do we need this? (e.g., "What happens if 1 million people try to login at once?")
		- **The Solution**: Explain the design step-by-step.
			- For HLD: Explain the flow clearly.
			- For LLD: Provide **commented, easy-to-read code snippets** (Go or Java).
		- **Why It Matters**: Practical takeaways for their future interviews or projects.
	3. **Formatting**: Markdown. Use triple backticks for code.
	
	SURPRISE ME. Pick a topic that is fundamental yet fascinating for a student.
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
