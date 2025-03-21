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
	// Получение DATABASE_URL из окружения
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Секретный ключ для JWT
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "mysecretkey"
	}

	// Повторные попытки подключения к базе данных
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
		log.Printf("Failed to connect to database: %v, retrying in 2 seconds...", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to database after retries:", err)
	}
	defer db.Close()

	// Применение миграций
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("Failed to create migration driver:", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		log.Fatal("Failed to initialize migrations:", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Failed to apply migrations:", err)
	}

	// Инициализация репозитория и хэндлеров
	repo := repository.NewNotesRepository(db)
	handler := handlers.NewNotesHandler(repo)

	// Настройка Gin
	r := gin.Default()

	// Публичные маршруты (без JWT)
	r.POST("/register", handler.Register)
	r.POST("/login", func(c *gin.Context) {
		c.Set("jwtSecret", jwtSecret) // Передаём секрет в контекст для Login
		handler.Login(c)
	})

	// Защищённые маршруты (с JWT)
	authGroup := r.Group("/", handlers.AuthMiddleware(jwtSecret))
	{
		authGroup.POST("/notes", handler.CreateNote)
		authGroup.PATCH("/notes/:id/delete", handler.DeleteNote)
		authGroup.PATCH("/notes/:id/restore", handler.RestoreNote)
		authGroup.GET("/notes/:id", handler.GetNote)
		authGroup.GET("/notes", handler.GetNotes)
	}

	// Запуск сервера
	r.Run(":8080")
}
