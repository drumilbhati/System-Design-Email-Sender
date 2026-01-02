package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/drumil/system-design-mailer/internal/ai"
	"github.com/drumil/system-design-mailer/internal/config"
	"github.com/drumil/system-design-mailer/internal/mailer"
	"github.com/drumil/system-design-mailer/internal/scheduler"
	"github.com/drumil/system-design-mailer/internal/store"
)

func main() {
	// 0. Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// 1. Load Config
	cfg := config.Load()

	// 2. Initialize Store
	var subStore store.Store
	var err error

	if mongoURI := os.Getenv("MONGO_URI"); mongoURI != "" {
		log.Println("Initializing MongoDB store...")
		subStore, err = store.NewMongoStore(mongoURI)
	} else {
		log.Println("Initializing File store (local)...")
		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "."
		}
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Printf("Warning: could not create data directory: %v", err)
		}
		storePath := fmt.Sprintf("%s/subscribers.json", dataDir)
		subStore, err = store.NewFileStore(storePath)
	}

	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	// 3. Initialize AI
	aiClient, err := ai.NewContentGenerator(cfg.GeminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize AI client: %v", err)
	}
	defer aiClient.Close()

	// 4. Initialize Mailer (Gmail API)
	// We read credentials from ENV or file
	credsJSON := os.Getenv("GMAIL_CREDENTIALS_JSON")
	if credsJSON == "" {
		// Fallback to local file for dev
		b, err := os.ReadFile("credentials.json")
		if err == nil {
			credsJSON = string(b)
		}
	}

	var emailSender interface {
		Send(to []string, subject, bodyHTML string) error
	}

	if credsJSON != "" {
		log.Println("Initializing Gmail API Mailer...")
		gm, err := mailer.NewGmailMailer(context.Background(), cfg.SenderEmail, []byte(credsJSON))
		if err != nil {
			log.Fatalf("Failed to create Gmail client: %v", err)
		}
		emailSender = gm
	} else {
		// Fallback to SMTP (will likely fail on Render, but keeps local dev simple if needed)
		log.Println("No Gmail credentials found. Falling back to SMTP (Legacy)...")
		emailSender = mailer.NewSMTPMailer(
			cfg.SMTPHost,
			cfg.SMTPPort,
			cfg.SMTPUser,
			cfg.SMTPPass,
			cfg.SenderEmail,
		)
	}

	// 5. Define the Daily Job
	dailyJob := func() {
		log.Println("Starting daily newsletter generation...")
		
		subscribers, err := subStore.GetAll()
		if err != nil {
			log.Printf("Error fetching subscribers: %v", err)
			return
		}

		if len(subscribers) == 0 {
			log.Println("No subscribers to send to. Skipping.")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		log.Println("Generating content with Gemini...")
		articleHTML, err := aiClient.GenerateArticle(ctx)
		if err != nil {
			log.Printf("Error generating article: %v", err)
			return
		}

		subject := "Daily System Design Article - " + time.Now().Format("Jan 02, 2006")
		
		log.Printf("Sending email to %d subscribers...", len(subscribers))
		err = emailSender.Send(subscribers, subject, articleHTML)
		if err != nil {
			log.Printf("Error sending emails: %v", err)
		} else {
			log.Println("Daily newsletter sent successfully!")
		}
	}

	// 6. Start Scheduler (24 hours)
	// For demo purposes, let's trigger it 10 seconds after start, then every 24 hours?
	// Or strictly 24 hours. The prompt says "everyday".
	// Let's stick to strict 24h interval.
	jobScheduler := scheduler.NewScheduler(24*time.Hour, dailyJob)
	jobScheduler.Start()

	// 7. HTTP Server for Subscriptions
	// Serve static files (Frontend)
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	http.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Email == "" || !isValidEmail(req.Email) {
			http.Error(w, "Invalid email address", http.StatusBadRequest)
			return
		}

		if err := subStore.Add(req.Email); err != nil {
			log.Printf("Failed to add subscriber: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Subscribed %s successfully", req.Email)
		log.Printf("New subscriber: %s", req.Email)
	})
	
	// Manual trigger endpoint for testing
	http.HandleFunc("/trigger-now", func(w http.ResponseWriter, r *http.Request) {
		// Basic security check (in real app, use auth)
		inputKey := r.URL.Query().Get("key")
		if cfg.CronSecret == "" {
			log.Println("Error: CRON_SECRET is not set. Manual trigger disabled.")
			http.Error(w, "Configuration error", http.StatusInternalServerError)
			return
		}
		if inputKey != cfg.CronSecret {
			log.Println("Auth failed: Invalid key provided")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		go dailyJob()
		log.Println("Manual trigger received. Starting daily job...")
		w.Write([]byte("Job triggered manually"))
	})

	srv := &http.Server{Addr: ":" + cfg.Port}

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down...")

	jobScheduler.Stop()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func isValidEmail(email string) bool {
	// Simple regex for email validation
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return re.MatchString(email)
}
