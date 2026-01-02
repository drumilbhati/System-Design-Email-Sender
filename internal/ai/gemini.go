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

func (c *ContentGenerator) GenerateArticle(ctx context.Context, overrideInstruction string) (string, error) {
	prompt := `
	You are a Senior Mentor and Technical Lead writing a daily educational newsletter for **college students and junior engineers**.
	
	Your goal is to explain complex software engineering concepts in a way that is **accessible, encouraging, and easy to understand**, while still being technically accurate. Avoid overly dense jargon; if you use a complex term, explain it simply first.
	
	Your task is to generate an article on a **randomly selected topic** from one of the following categories. Pick ONE category and one specific topic.

	### CATEGORY 1: Core Distributed Systems Concepts
	Explain a fundamental concept that powers modern systems.
	- Examples: "Consistent Hashing", "CAP Theorem", "Load Balancing Algorithms", "Database Sharding vs Partitioning", "Raft Consensus (Simplified)", "Bloom Filters".
	- Focus on: **Why do we need this?** (The problem it solves) and how it works conceptually.

	### CATEGORY 2: High-Level System Design (HLD)
	Architect a familiar application.
	- Examples: "Design a URL Shortener", "Design Instagram's Feed", "Design a Chat Application", "Design a Rate Limiter".
	- Focus on: The high-level components (DB, Cache, Server) and how data flows between them.

	### CATEGORY 3: Low-Level Design (LLD), Patterns & SOLID
	Zoom in on coding patterns, object-oriented design, and SOLID principles.
	- Examples: "Understanding the Single Responsibility Principle", "Factory Pattern vs Abstract Factory", "Implementing an LRU Cache", "Thread Pools explained", "Observer Pattern in Real Life".
	- Focus on: Clean code examples, class diagrams (text-based), and **why** a pattern is used.

	### GUIDELINES:
	1. **Tone**: **"The Smart Senior Student"**. Explain it like you are teaching a friend in the college library.
		- **Simplify Complexity**: If explaining Raft or Consistent Hashing, DO NOT dump math. Use analogies (e.g., "Imagine a ring of servers..." or "Think of consensus like a group voting on where to eat lunch").
	2. **Structure**:
		- **Title**: Clear and descriptive.
		- **The "Why"**: Start with a simple problem statement. (e.g., "Why does a standard hash function fail when we add a server?")
		- **The "How" (Concept)**: Explain the solution using simple terms and diagrams (described in text).
		- **Code / Architecture**: Show the structure. For LLD, provide **commented code** (Java or Go).
		- **Real World**: Where is this actually used? (e.g., "DynamoDB uses this").
	3. **Formatting**: Markdown. Use triple backticks for code.
	
	SURPRISE ME. Pick a topic that makes the student go "Oh, so THAT is how it works!".
	`

	if overrideInstruction != "" {
		prompt += fmt.Sprintf("\n\n**IMPORTANT OVERRIDE**: %s", overrideInstruction)
	}

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
