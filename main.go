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

	// Повторные попытки подключения к базе данных
	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ { // Пробуем 10 раз с интервалом 2 секунды
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			err = db.Ping() // Проверяем доступность базы
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
	r.POST("/notes", handler.CreateNote)
	r.PATCH("/notes/:id/delete", handler.DeleteNote)
	r.PATCH("/notes/:id/restore", handler.RestoreNote)
	r.GET("/notes/:id", handler.GetNote)
	r.GET("/notes", handler.GetNotes)

	// Запуск сервера
	r.Run(":8080")
}
