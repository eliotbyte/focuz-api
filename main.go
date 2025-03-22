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
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "mysecretkey"
	}

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

	spacesRepo := repository.NewSpacesRepository(db)
	topicsRepo := repository.NewTopicsRepository(db)
	notesRepo := repository.NewNotesRepository(db)
	rolesRepo := repository.NewRolesRepository(db)

	notesHandler := handlers.NewNotesHandler(notesRepo, spacesRepo, topicsRepo)
	spacesHandler := handlers.NewSpacesHandler(spacesRepo, rolesRepo)
	topicsHandler := handlers.NewTopicsHandler(topicsRepo, spacesRepo)

	r := gin.Default()

	r.POST("/register", notesHandler.Register)
	r.POST("/login", func(c *gin.Context) {
		c.Set("jwtSecret", jwtSecret)
		notesHandler.Login(c)
	})

	auth := r.Group("/", handlers.AuthMiddleware(jwtSecret))
	{
		auth.GET("/spaces", spacesHandler.GetAccessibleSpaces)
		auth.DELETE("/spaces/:id/users/:userId", spacesHandler.RemoveUser)

		auth.POST("/spaces", spacesHandler.CreateSpace)
		auth.PATCH("/spaces/:id", spacesHandler.UpdateSpace)
		auth.PATCH("/spaces/:id/delete", spacesHandler.DeleteSpace)
		auth.PATCH("/spaces/:id/restore", spacesHandler.RestoreSpace)
		auth.POST("/spaces/:id/invite", spacesHandler.InviteUser)

		auth.POST("/topics", topicsHandler.CreateTopic)
		auth.PATCH("/topics/:id", topicsHandler.UpdateTopic)
		auth.PATCH("/topics/:id/delete", topicsHandler.DeleteTopic)
		auth.PATCH("/topics/:id/restore", topicsHandler.RestoreTopic)
		auth.GET("/spaces/:spaceId/topics", topicsHandler.GetTopicsBySpace)

		auth.POST("/notes", notesHandler.CreateNote)
		auth.PATCH("/notes/:id/delete", notesHandler.DeleteNote)
		auth.PATCH("/notes/:id/restore", notesHandler.RestoreNote)
		auth.GET("/notes/:id", notesHandler.GetNote)
		auth.GET("/notes", notesHandler.GetNotes)
	}

	r.Run(":8080")
}
