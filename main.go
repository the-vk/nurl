package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	PROJECT_ID = "nurl-458720"
	DATABASE   = "nurl"
	BASE_URL   = "http://localhost:8080/"
)

type UrlRecord struct {
	Short     string    `firestore:"short"`
	Long      string    `firestore:"long"`
	CreatedAt time.Time `firestore:"created_at"`
}

func main() {
	fmt.Println("Starting HTTP server on port 8080...")

	// Register handlers
	http.HandleFunc("/health-check", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			longUrl := r.URL.Query().Get("url")
			shortUrl, err := store(longUrl)
			if err != nil {
				fmt.Println("Failed to store key %s: %s", longUrl, err)
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Shortened \"%s\" to \"%s%s\"\n", longUrl, BASE_URL, *shortUrl)

		case http.MethodGet:
			shortUrl := strings.TrimPrefix(r.URL.Path, "/")
			record, err := query(shortUrl)
			if err != nil {
				fmt.Println("Failed to query short url %s: %s", shortUrl, err)
			}
			if record == nil {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.Header().Set("Location", record.Long)
				w.WriteHeader(http.StatusMovedPermanently)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Start the server
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Server error: %s\n", err)
	}
}

func store(longUrl string) (*string, error) {
	canonicalUrl, err := url.Parse(longUrl)
	if err != nil {
		return nil, fmt.Errorf("url: %s is not a valid URL", longUrl)
	}

	ctx := context.Background()
	client, err := firestore.NewClientWithDatabase(ctx, PROJECT_ID, DATABASE)
	if err != nil {
		return nil, err
	}

	collection := client.Collection("Urls")
	existingRecordQuery := collection.Where("long", "==", canonicalUrl.String())
	iter := existingRecordQuery.Documents(ctx)
	existingRecords, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	var record UrlRecord
	if len(existingRecords) != 0 {
		existingRecords[0].DataTo(&record)
		return &record.Short, nil
	}

	for {
		shortUrl := shorten(longUrl)
		record = UrlRecord{Short: shortUrl, Long: longUrl, CreatedAt: time.Now()}
		_, _, err := collection.Add(ctx, record)
		if err == nil {
			break
		}
	}

	return &record.Short, nil
}

func query(shortUrl string) (*UrlRecord, error) {
	ctx := context.Background()
	client, err := firestore.NewClientWithDatabase(ctx, PROJECT_ID, DATABASE)
	if err != nil {
		return nil, err
	}

	collection := client.Collection("Urls")
	q := collection.Where("short", "==", shortUrl).Limit(1)
	records, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	var record UrlRecord
	records[0].DataTo(&record)
	return &record, nil
}

func shorten(originalUrl string) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 7

	// Create a new random source with current time as seed
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}

	return string(result)
}
