package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/SOMAK939/file-sharing-platform/config"
	"github.com/SOMAK939/file-sharing-platform/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	
	"github.com/SOMAK939/file-sharing-platform/workers"
)

func main() {

	fmt.Println("Loaded JWT_SECRET:", os.Getenv("JWT_SECRET"))

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("  Warning: No .env file found")
	}
    
	// Initialize Redis
	config.InitRedis()

	// Connect to PostgreSQL
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(" Unable to connect to database:", err)
	}
	defer db.Close()

	// Verify DB connection
	if err := db.Ping(); err != nil {
		log.Fatal(" Database ping failed:", err)
	} else {
		fmt.Println(" Connected to PostgreSQL!")
	}

		

	// Upload queue for concurrency
	uploadQueue := make(chan string, 10)
	go handlers.ProcessUploads(uploadQueue)

	// Set up router
	router := mux.NewRouter()
	router.HandleFunc("/register", handlers.RegisterUser(db)).Methods("POST")
	router.HandleFunc("/login", handlers.LoginUser(db)).Methods("POST")
	router.HandleFunc("/upload", handlers.UploadFile(db, uploadQueue)).Methods("POST")

	router.HandleFunc("/download/{filename}", handlers.DownloadFile).Methods("GET")
	router.HandleFunc("/file/{filename}", handlers.GetFileURL(db)).Methods("GET")
	router.HandleFunc("/share/{file_id}", handlers.GetFileShareableURL(db)).Methods("GET")
	router.HandleFunc("/user/files", handlers.GetUserFiles(db, config.RDB)).Methods("GET")
	router.HandleFunc("/search", handlers.SearchFiles(db, config.RDB)).Methods("GET")
	router.HandleFunc("/files/{file_id}", handlers.GetFileMetadata(db, config.RDB)).Methods("GET")
	router.HandleFunc("/files/{file_id}/rename", handlers.RenameFile(db, config.RDB)).Methods("PUT")
	router.HandleFunc("/ws", handlers.WebSocketHandler)


	// Start background worker for expired file cleanup
    workers.StartFileCleanupWorker(db)
	


	// Start server
	fmt.Println(" Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}
