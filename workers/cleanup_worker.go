package workers

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/SOMAK939/file-sharing-platform/utils"
)

// StartFileCleanupWorker runs a background job for expired file deletion
func StartFileCleanupWorker(db *sql.DB) {
	fmt.Println("✅ Starting Background Cleanup Worker...") // ADD THIS
	ticker := time.NewTicker(1 * time.Hour) // Runs every 1 hour
	go func() {
		for range ticker.C {
			log.Println("🧹 Running file cleanup job...")
			err := deleteExpiredFiles(db)
			if err != nil {
				log.Println("❌ File cleanup job failed:", err)
			}
			
		}
	}()
}

// deleteExpiredFiles finds and removes expired files
func deleteExpiredFiles(db *sql.DB) error {
	expirationThreshold := time.Now().Add(-1*time.Hour) // Files older than 1 Hour

	rows, err := db.Query("SELECT id, COALESCE(file_url, '') FROM files WHERE uploaded_at < $1", expirationThreshold)
	if err != nil {
		return fmt.Errorf("error fetching expired files: %v", err)
	}
	defer rows.Close()

	var expiredFiles []struct {
		ID      int
		FileURL string
	}

	for rows.Next() {
		var file struct {
			ID      int
			FileURL string
		}
		if err := rows.Scan(&file.ID, &file.FileURL); err != nil {
			log.Printf("⚠️ Skipping file due to scan error: %v\n", err)
			continue // Continue processing other rows instead of stopping
		}
		expiredFiles = append(expiredFiles, file)
	}

	for _, file := range expiredFiles {
		if file.FileURL == "" {
			log.Printf("⚠️ Skipping file ID %d because it has an empty file_url\n", file.ID)
			continue
		}

		err := utils.DeleteFromS3(file.FileURL)
		if err != nil {
			log.Printf("❌ Failed to delete file from S3 (%s): %v\n", file.FileURL, err)
			continue
		}

		_, err = db.Exec("DELETE FROM files WHERE id = $1", file.ID)
		if err != nil {
			log.Printf("❌ Failed to delete file record (%d): %v\n", file.ID, err)
		} else {
			log.Printf("✅ Deleted expired file: %s\n", file.FileURL)
		}
	}

	return nil
}
