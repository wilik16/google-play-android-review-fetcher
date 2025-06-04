package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

// Configuration
var (
	packageName string
	webhookURL  string
	keyPath     string
	reviewsCSV  string
	logDir      string
	testMode    bool
)

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// Review represents a single review from the Play Store
type Review struct {
	ReviewID    string    `json:"reviewId"`
	Rating      int       `json:"rating"`
	Text        string    `json:"text"`
	Author      string    `json:"author"`
	Device      string    `json:"device"`
	Date        time.Time `json:"date"`
	Notified    bool      `json:"notified"`
}

// RatingRange represents a range of ratings to filter
type RatingRange struct {
	Start int
	End   int
}

// setupLogging initializes the logging system
func setupLogging() (*os.File, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	logFile, err := os.OpenFile(
		filepath.Join(logDir, fmt.Sprintf("reviews-%s.log", today)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	log.SetOutput(logFile)
	return logFile, nil
}

// parseRatingRange parses the rating range from command line arguments
func parseRatingRange() (*RatingRange, error) {
	if len(os.Args) < 2 {
		return nil, nil
	}

	parts := strings.Split(os.Args[1], "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid rating range format. Use format: start-end (e.g., 1-3)")
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid start rating: %v", err)
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid end rating: %v", err)
	}

	if start < 1 || start > 5 || end < 1 || end > 5 {
		return nil, fmt.Errorf("rating range must be between 1 and 5")
	}

	if start > end {
		return nil, fmt.Errorf("start rating must be less than or equal to end rating")
	}

	return &RatingRange{Start: start, End: end}, nil
}

// readExistingReviews reads existing reviews from CSV file
func readExistingReviews() (map[string]Review, error) {
	reviews := make(map[string]Review)

	if _, err := os.Stat(reviewsCSV); os.IsNotExist(err) {
		return reviews, nil
	}

	file, err := os.Open(reviewsCSV)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	// Skip header
	for _, record := range records[1:] {
		date, _ := time.Parse(time.RFC3339, record[5])
		notified, _ := strconv.ParseBool(record[6])
		rating, _ := strconv.Atoi(record[1])

		reviews[record[0]] = Review{
			ReviewID: record[0],
			Rating:   rating,
			Text:     record[2],
			Author:   record[3],
			Device:   record[4],
			Date:     date,
			Notified: notified,
		}
	}

	return reviews, nil
}

// saveReviews saves reviews to CSV file
func saveReviews(reviews []Review) error {
	file, err := os.Create(reviewsCSV)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Review ID", "Rating", "Review Text", "Author", "Device", "Date", "Notified"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %v", err)
	}

	// Write records
	for _, review := range reviews {
		record := []string{
			review.ReviewID,
			strconv.Itoa(review.Rating),
			review.Text,
			review.Author,
			review.Device,
			review.Date.Format(time.RFC3339),
			strconv.FormatBool(review.Notified),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %v", err)
		}
	}

	return nil
}

// sendToWebhook sends a review to the webhook
func sendToWebhook(review Review) error {
	stars := strings.Repeat("⭐", review.Rating)
	message := map[string]string{
		"text": fmt.Sprintf("*New Review*\nRating: %s (%d)\nReview: %s\nAuthor: %s\nDevice: %s\nDate: %s",
			stars, review.Rating, review.Text, review.Author, review.Device, review.Date.Format(time.RFC1123)),
	}

	if testMode {
		log.Printf("TEST MODE - Would send to webhook:\n%s", message["text"])
		time.Sleep(2 * time.Second)
		return nil
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	resp, err := http.Post(webhookURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned non-200 status code: %d", resp.StatusCode)
	}

	// Add delay to avoid rate limiting
	time.Sleep(2 * time.Second)
	return nil
}

// cleanText removes leading and trailing whitespace and tabs from text
func cleanText(text string) string {
	return strings.TrimSpace(text)
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	} else {
		log.Printf("Successfully loaded .env file")
	}

	// Initialize configuration
	packageName = getEnv("PACKAGE_NAME", "com.example.app")
	webhookURL = getEnv("WEBHOOK_URL", "https://chat.googleapis.com/v1/spaces/XXXXX/messages?key=YOUR_KEY&token=YOUR_TOKEN")
	keyPath = getEnv("KEY_PATH", "./service-account.json")
	reviewsCSV = getEnv("REVIEWS_CSV", "reviews.csv")
	logDir = getEnv("LOG_DIR", "./logs")
	testMode = getEnvBool("TEST_MODE", true)

	// Setup logging
	logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}
	defer logFile.Close()

	// Parse rating range
	ratingRange, err := parseRatingRange()
	if err != nil {
		log.Fatalf("Failed to parse rating range: %v", err)
	}
	if ratingRange != nil {
		log.Printf("Filtering reviews with ratings from %d to %d", ratingRange.Start, ratingRange.End)
	}

	// Initialize Google API client
	ctx := context.Background()
	credentials, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("Failed to read credentials file: %v", err)
	}

	config, err := google.JWTConfigFromJSON(credentials, androidpublisher.AndroidpublisherScope)
	if err != nil {
		log.Fatalf("Failed to create JWT config: %v", err)
	}

	service, err := androidpublisher.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
	if err != nil {
		log.Fatalf("Failed to create Android Publisher service: %v", err)
	}

	// Get existing reviews
	existingReviews, err := readExistingReviews()
	if err != nil {
		log.Fatalf("Failed to read existing reviews: %v", err)
	}
	log.Printf("Found %d existing reviews in CSV", len(existingReviews))

	// Fetch new reviews
	reviews, err := service.Reviews.List(packageName).Do()
	if err != nil {
		log.Printf("Failed to fetch reviews: %v", err)
		log.Printf("packageName: %s", packageName)
		log.Fatalf("Failed to fetch reviews: %v", err)
	}
	log.Printf("Total reviews fetched from API: %d", len(reviews.Reviews))

	// Process new reviews
	var newReviews []Review
	for _, review := range reviews.Reviews {
		rating := int(review.Comments[0].UserComment.StarRating)
		if ratingRange != nil && (rating < ratingRange.Start || rating > ratingRange.End) {
			continue
		}

		var device = "Unknown"
		if review.Comments[0].UserComment.DeviceMetadata != nil {
			device = review.Comments[0].UserComment.DeviceMetadata.ProductName
		}

		if existingReview, exists := existingReviews[review.ReviewId]; !exists || !existingReview.Notified {
			newReview := Review{
				ReviewID: review.ReviewId,
				Rating:   rating,
				Text:     cleanText(review.Comments[0].UserComment.Text),
				Author:   review.AuthorName,
				Device:   device,
				Date:     time.Unix(review.Comments[0].UserComment.LastModified.Seconds, 0),
				Notified: false,
			}

			if err := sendToWebhook(newReview); err != nil {
				log.Printf("Failed to send review %s to webhook: %v", review.ReviewId, err)
				continue
			}

			newReview.Notified = true
			newReviews = append(newReviews, newReview)
			existingReviews[review.ReviewId] = newReview
		}
	}

	// Save all reviews
	var allReviews []Review
	for _, review := range existingReviews {
		allReviews = append(allReviews, review)
	}
	if err := saveReviews(allReviews); err != nil {
		log.Fatalf("Failed to save reviews: %v", err)
	}

	// Log results
	if len(newReviews) > 0 {
		log.Printf("\n=== New Reviews ===")
		for i, review := range newReviews {
			stars := strings.Repeat("⭐", review.Rating)
			log.Printf("[%d] %s | %s | %s | %s | %s",
				i+1, stars, review.Text, review.Author, review.Device, review.Date.Format(time.RFC1123))
		}
	} else {
		log.Println("No new reviews found")
	}
} 