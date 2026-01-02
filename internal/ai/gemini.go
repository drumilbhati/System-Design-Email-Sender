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
	You are a Senior Staff Software Engineer writing a daily technical newsletter for other experienced engineers.
	
	Your task is to generate a deep, high-quality technical article on a **randomly selected topic** from one of the following three categories. Pick ONE category and one specific topic within it. Do not announce the category, just dive into the topic.

	### CATEGORY 1: Real-World System Breakdowns (Case Studies)
	Analyze how a major tech company solved a specific scaling problem. Base this on common public engineering challenges (e.g., from blogs like Uber, Netflix, Meta, Discord, Stripe).
	- Examples: "How Discord stores billions of messages (Cassandra to ScyllaDB)", "Uber's Ringpop for distributed state", "Netflix's chaos engineering principles", "Instagram's ID generation with Postgres".
	- Focus on: The specific problem, the architectural evolution, and the trade-offs.

	### CATEGORY 2: High-Level Design (HLD)
	Design a complex distributed system component.
	- Examples: "Designing a Distributed Job Scheduler", "Architecture of a Real-time Collaborative Editor (OT vs CRDT)", "Building a Geo-Spatial Index for Proximity Search", "Design of a Write-Heavy Analytics System".
	- Focus on: Data flow, database choice (SQL vs NoSQL), caching strategies, consistency models (CAP theorem application), and failure scenarios.

	### CATEGORY 3: Low-Level Design (LLD) & Internals
	Deep dive into code, algorithms, or language internals (specifically Go context).
	- Examples: "Implementing a Lock-Free Ring Buffer", "How Go's Garbage Collector actually works", "Writing a Custom Memory Allocator", "Database Internals: LSM Trees vs B-Trees implementation details".
	- Focus on: Concurrency patterns, memory management, performance optimization, and idiomatic Go code.

	### GUIDELINES:
	1. **Tone**: Professional, "Engineer-to-Engineer". No fluff, no "In this fast-paced world" intros. Jump straight into the technical meat.
	2. **Structure**:
		- **Title**: Catchy and technical.
		- **The Problem**: What are we solving? Why is it hard?
		- **The Solution (Architecture/Code)**:
			- For HLD: Explain components, diagrams (described in text), and data flow.
			- For LLD: Provide **idiomatic code snippets** (Java or Go) demonstrating the core logic.
		- **War Stories / Trade-offs**: What breaks? What are the limitations?
		- **Key Takeaways**: Bullet points.
	3. **Formatting**: Markdown. Use triple backticks for code.
	
	SURPRISE ME. Do not always pick the most popular topic. Explore niche but critical engineering concepts.
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
