package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
     "strings"
	 

  awsConfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
	

	
	appConfig "github.com/SOMAK939/file-sharing-platform/config" 

	

	"github.com/gorilla/mux"
)

// S3 Client Initialization
var s3Client *s3.Client
var bucketName = os.Getenv("AWS_S3_BUCKET_NAME")

// Initialize AWS S3 Client
func InitS3() {
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}
	s3Client = s3.NewFromConfig(cfg)
}

// FileMetadata struct
type FileMetadata struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	Filepath string `json:"filepath"`
	URL      string `json:"url"`
}

// GetFileURL retrieves file metadata and provides a downloadable link
func GetFileURL(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileID := vars["file_id"]

		var fileMeta FileMetadata
		err := db.QueryRow("SELECT id, filename, filepath FROM files WHERE id=$1", fileID).
			Scan(&fileMeta.ID, &fileMeta.Filename, &fileMeta.Filepath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Construct public download URL
		publicURL := fmt.Sprintf("http://localhost:8080/download/%s", fileMeta.Filename)
		fileMeta.URL = publicURL

		// Send JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fileMeta)
	}
}

// DownloadFile serves the file for download
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileName := vars["filename"]
	filePath := filepath.Join("uploads", fileName)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Get file info
	fileStat, err := file.Stat()
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Disposition", "attachment; filename="+fileStat.Name())
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileStat.Size()))

	// Stream file to response
	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Error streaming file", http.StatusInternalServerError)
	}
}

// UploadFile handles file upload and metadata storage
func UploadFile(db *sql.DB, uploadQueue chan string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get JWT token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "❌ Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}

		// Extract token (Bearer Token)
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := appConfig.ValidateJWT(tokenStr)
		if err != nil {
			fmt.Println("JWT Validation Error:", err) // 🔍 Debugging line
			http.Error(w, "❌ Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}
		fmt.Println("Extracted User ID:", userID) // 🔍 Debugging line


		// Parse the file from request
		err = r.ParseMultipartForm(10 << 20) // 10MB max
		if err != nil {
			http.Error(w, "File too large", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Generate unique filename
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
		filePath := filepath.Join("uploads", filename)

		// Save locally
		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Could not create file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		_, err = io.Copy(dst, file)

		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

			// Start a database transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "❌ Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() // Rollback in case of failure

	// Upload to S3
	s3URL, err := UploadToS3(file, filename)
	if err != nil {
		http.Error(w, "S3 upload failed", http.StatusInternalServerError)
		return
	}

	// Store metadata in DB using transaction
	_, err = tx.Exec("INSERT INTO files (filename, filepath, uploaded_at, file_url, owner_id) VALUES ($1, $2, $3, $4, $5)",
		filename, filePath, time.Now(), s3URL, userID)

	if err != nil {
		log.Println("❌ Database insert error:", err) // Log the actual SQL error
		http.Error(w, "Database insert failed", http.StatusInternalServerError)
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		http.Error(w, "❌ Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Respond
	json.NewEncoder(w).Encode(map[string]string{
		"message": "✅ File uploaded successfully!",
		"url":     s3URL,
	})
	

	// Notify user via WebSocket
	go NotifyUploadComplete(filename, userID)


	}
}



func UploadToS3(file multipart.File, fileName string) (string, error) {
	// Load AWS Config
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Printf("❌ AWS Config Load Error: %v\n", err)
		return "", err
	}

	// Create S3 client
	svc := s3.NewFromConfig(cfg)

	// Upload file
	_, err = svc.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("AWS_S3_BUCKET_NAME")),
		Key:    aws.String(fileName),
		Body:   file,
		
	})
	if err != nil {
		log.Printf("❌ S3 Upload Error: %v\n", err)
		return "", err
	}

	// Return public S3 URL
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		os.Getenv("AWS_S3_BUCKET_NAME"),
		os.Getenv("AWS_REGION"),
		fileName)
	
	log.Printf("✅ File uploaded to S3: %s\n", fileURL)
	return fileURL, nil
}


// ProcessUploads handles background file processing
func ProcessUploads(uploadQueue chan string) {
	for filePath := range uploadQueue {
		fmt.Println("Processing uploaded file:", filePath)
		time.Sleep(2 * time.Second) // Simulating processing time
		fmt.Println("File processed successfully:", filePath)
	}
}

// GetFileShareableURL generates a public URL for a file
func GetFileShareableURL(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileID := vars["file_id"]

		var fileURL string
		err := db.QueryRow("SELECT file_url FROM files WHERE id = $1", fileID).Scan(&fileURL)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"shareable_url": fileURL})
	}
}

func GetUploadedFiles(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, filename, file_url, uploaded_at FROM files")
		if err != nil {
			log.Println("❌ Database query failed:", err)
			http.Error(w, "Database query failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var files []map[string]string

		for rows.Next() {
			var id int
			var filename string
			var fileURL sql.NullString // Handling NULL values properly
			var uploadedAt sql.NullTime

			if err := rows.Scan(&id, &filename, &fileURL, &uploadedAt); err != nil {
				log.Println("❌ Error scanning row:", err)
				http.Error(w, "Error scanning row", http.StatusInternalServerError)
				return
			}

			// Handle NULL values properly
			fileURLStr := "N/A"
			if fileURL.Valid {
				fileURLStr = fileURL.String
			}

			uploadedAtStr := "N/A"
			if uploadedAt.Valid {
				uploadedAtStr = uploadedAt.Time.Format(time.RFC3339)
			}

			files = append(files, map[string]string{
				"id":          fmt.Sprintf("%d", id),
				"filename":    filename,
				"url":         fileURLStr,
				"uploaded_at": uploadedAtStr,
			})
		}

		// Respond with JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}
}


func SearchFiles(db *sql.DB, RDB *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query == "" {
			http.Error(w, "query parameter is required", http.StatusBadRequest)
			return
		}

		cacheKey := fmt.Sprintf("search:%s", query) // Redis cache key

		// 1️⃣ Check Redis cache first
		cachedData, err := RDB.Get(context.Background(), cacheKey).Result()
		if err == nil {
			// If data exists in cache, return it
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(cachedData))
			return
		}

		// 2️⃣ If not cached, query the database
		rows, err := db.Query("SELECT id, filename, file_url FROM files WHERE filename ILIKE $1 OR file_url ILIKE $1", "%"+query+"%")
		if err != nil {
			http.Error(w, "Database query error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var results []FileMetadata
		for rows.Next() {
			var file FileMetadata
			if err := rows.Scan(&file.ID, &file.Filename, &file.URL); err != nil {
				log.Println("❌ Error scanning row:", err)
				continue
			}
			results = append(results, file)
		}

		// 3️⃣ Store results in Redis for future searches
		jsonData, _ := json.Marshal(results)
		RDB.Set(context.Background(), cacheKey, jsonData, 10*time.Minute) // Cache for 10 min

		// 4️⃣ Return JSON response
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

func RenameFile(db *sql.DB, RDB *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileID := vars["file_id"]

		// Get new filename from request body
		var requestBody struct {
			NewFilename string `json:"new_filename"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Update filename in DB
		_, err := db.Exec("UPDATE files SET filename = $1 WHERE id = $2", requestBody.NewFilename, fileID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Invalidate the cache
		cacheKey := "file_metadata:" + fileID
		RDB.Del(context.Background(), cacheKey)

		// Return success message
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("File renamed successfully and cache invalidated"))
	}
}


func GetFileMetadata(db *sql.DB, RDB *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileID := vars["file_id"]
		ctx := context.Background()

		// Check Redis cache first
		cacheKey := "file_metadata:" + fileID
		cachedMetadata, err := RDB.Get(ctx, cacheKey).Result()
		if err == nil {
			// Cache hit! Return cached data
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(cachedMetadata))
			return
		}

		// Cache miss! Query the database
		var file FileMetadata
		err = db.QueryRow("SELECT id, filename, filepath, file_url FROM files WHERE id = $1", fileID).
			Scan(&file.ID, &file.Filename, &file.Filepath, &file.URL)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Convert file metadata to JSON
		jsonMetadata, _ := json.Marshal(file)

		// Store metadata in Redis (expire in 5 minutes)
		RDB.Set(ctx, cacheKey, jsonMetadata, 5*time.Minute)

		// Return the metadata
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonMetadata)
	}
}

func GetUserFiles(db *sql.DB, RDB *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get JWT token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "❌ Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}

		// Extract token (Bearer Token)
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := appConfig.ValidateJWT(tokenStr)
		if err != nil {
			http.Error(w, "❌ Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// Check Redis cache
		cacheKey := fmt.Sprintf("user:files:%s", userID)
		cachedData, err := RDB.Get(context.TODO(), cacheKey).Result()
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(cachedData))
			return
		}

		// Fetch user files from DB
		rows, err := db.Query("SELECT id, filename, file_url FROM files WHERE owner_id = $1 ORDER BY id DESC", userID)
		if err != nil {
			http.Error(w, "❌ Database error", http.StatusInternalServerError)
			log.Println("❌ Database query error:", err)  // Debugging log
			return
		}
		defer rows.Close()

		var files []FileMetadata
		for rows.Next() {
			var file FileMetadata
			if err := rows.Scan(&file.ID, &file.Filename, &file.URL); err != nil {
				http.Error(w, "❌ Error scanning row", http.StatusInternalServerError)
				return
			}
			files = append(files, file)
		}

		// Convert result to JSON and cache it
		fileMetaJSON, _ := json.Marshal(files)
		RDB.Set(context.TODO(), cacheKey, fileMetaJSON, 5*time.Minute)

		w.Header().Set("Content-Type", "application/json")
		w.Write(fileMetaJSON)
	}
}


