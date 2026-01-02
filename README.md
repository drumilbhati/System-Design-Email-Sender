# System Design Daily Mailer

A Go application that uses Gemini AI to generate a daily system design article and emails it to subscribers.

## Features
- **Daily Content Generation**: Uses Gemini Pro to create unique articles.
- **Email Delivery**: Sends HTML emails using SMTP.
- **Subscription Management**: HTTP endpoint to subscribe users.
- **Persistent Storage**: Saves subscribers to a JSON file.
- **Graceful Shutdown**: Handles OS signals properly.

## Setup

1. **Clone the repository**
2. **Set up Environment Variables**:
   Copy `.env.example` to `.env` (or set them in your shell).
   ```bash
   export GEMINI_API_KEY="your_key"
   export SMTP_HOST="smtp.gmail.com"
   export SMTP_PORT=587
   export SMTP_USER="your@email.com"
   export SMTP_PASS="your_password"
   export SENDER_EMAIL="your@email.com"
   ```

3. **Run the Application**:
   ```bash
   go run cmd/server/main.go
   ```

## Usage

- **Subscribe a user**:
  ```bash
  curl -X POST -d '{"email":"user@example.com"}' http://localhost:8080/subscribe
  ```

- **Trigger manually (for testing)**:
  ```bash
  curl "http://localhost:8080/trigger-now?key=your_smtp_password"
  ```
