package main

import (
	"database/sql"
	"focuz-api/handlers"
	"focuz-api/repository"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	// Get environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "mysecretkey"
	}

	// Retry database connection
	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("DB connection failed: %v, retrying in 2s...", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Could not connect to database:", err)
	}
	defer db.Close()

	// Run migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("Migration driver error:", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatal("Migration init error:", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Migration failed:", err)
	}

	// Setup repository and handlers
	repo := repository.NewNotesRepository(db)
	handler := handlers.NewNotesHandler(repo)

	// Setup Gin routes
	r := gin.Default()

	// Public routes
	r.POST("/register", handler.Register)
	r.POST("/login", func(c *gin.Context) {
		c.Set("jwtSecret", jwtSecret)
		handler.Login(c)
	})

	// Protected routes
	auth := r.Group("/", handlers.AuthMiddleware(jwtSecret))
	{
		auth.POST("/notes", handler.CreateNote)
		auth.PATCH("/notes/:id/delete", handler.DeleteNote)
		auth.PATCH("/notes/:id/restore", handler.RestoreNote)
		auth.GET("/notes/:id", handler.GetNote)
		auth.GET("/notes", handler.GetNotes)
	}

	// Start server
	r.Run(":8080")
}
