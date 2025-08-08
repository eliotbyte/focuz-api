package main

import (
	"database/sql"
	"focuz-api/handlers"
	"focuz-api/initializers"
	"focuz-api/middleware"
	"focuz-api/repository"
	"log"
	"os"
	"strings"
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

	if err := initializers.InitDefaults(db); err != nil {
		log.Fatal("Failed to initialize default data:", err)
	}

	if err := initializers.InitMinio(); err != nil {
		log.Fatal("Failed to initialize Minio:", err)
	}

	spacesRepo := repository.NewSpacesRepository(db)
	topicsRepo := repository.NewTopicsRepository(db)
	notesRepo := repository.NewNotesRepository(db)
	rolesRepo := repository.NewRolesRepository(db)
	activityTypesRepo := repository.NewActivityTypesRepository(db)
	activitiesRepo := repository.NewActivitiesRepository(db)
	attachmentsRepo := repository.NewAttachmentsRepository(db)
	chartsRepo := repository.NewChartsRepository(db)

	notesHandler := handlers.NewNotesHandler(notesRepo, spacesRepo, topicsRepo)
	spacesHandler := handlers.NewSpacesHandler(spacesRepo, rolesRepo)
	topicsHandler := handlers.NewTopicsHandler(topicsRepo, spacesRepo, rolesRepo)
	activityTypesHandler := handlers.NewActivityTypesHandler(activityTypesRepo, spacesRepo)
	activitiesHandler := handlers.NewActivitiesHandler(
		activitiesRepo,
		spacesRepo,
		topicsRepo,
		notesRepo,
		activityTypesRepo,
	)
	attachmentsHandler := handlers.NewAttachmentsHandler(attachmentsRepo, notesRepo, spacesRepo, topicsRepo)
	chartsHandler := handlers.NewChartsHandler(chartsRepo, spacesRepo, topicsRepo, activityTypesRepo)

	// Set Gin to release mode in production
	if os.Getenv("GIN_MODE") == "release" || strings.ToLower(os.Getenv("APP_ENV")) == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Configure trusted proxies for correct client IP handling in production
	trustedProxies := os.Getenv("TRUSTED_PROXIES")
	if trustedProxies != "" {
		parts := strings.Split(trustedProxies, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if err := r.SetTrustedProxies(parts); err != nil {
			log.Fatalf("Invalid TRUSTED_PROXIES: %v", err)
		}
	} else {
		// Default to loopback only; override via TRUSTED_PROXIES in production
		_ = r.SetTrustedProxies([]string{"127.0.0.1"})
	}

	r.Use(middleware.CORSMiddleware())

	r.POST("/register", notesHandler.Register)
	r.POST("/login", func(c *gin.Context) {
		c.Set("jwtSecret", jwtSecret)
		notesHandler.Login(c)
	})

	auth := r.Group("/", handlers.AuthMiddleware(jwtSecret))
	{
		auth.GET("/spaces", spacesHandler.GetAccessibleSpaces)
		auth.DELETE("/spaces/:spaceId/users/:userId", spacesHandler.RemoveUser)
		auth.GET("/spaces/:spaceId/users", spacesHandler.GetUsersInSpace)
		auth.POST("/spaces", spacesHandler.CreateSpace)
		auth.PATCH("/spaces/:spaceId", spacesHandler.UpdateSpace)
		auth.PATCH("/spaces/:spaceId/delete", spacesHandler.DeleteSpace)
		auth.PATCH("/spaces/:spaceId/restore", spacesHandler.RestoreSpace)
		auth.POST("/spaces/:spaceId/invite", spacesHandler.InviteUser)

		auth.POST("/topics", topicsHandler.CreateTopic)
		auth.PATCH("/topics/:id", topicsHandler.UpdateTopic)
		auth.PATCH("/topics/:id/delete", topicsHandler.DeleteTopic)
		auth.PATCH("/topics/:id/restore", topicsHandler.RestoreTopic)
		auth.GET("/spaces/:spaceId/topics", topicsHandler.GetTopicsBySpace)
		auth.GET("/topic-types", topicsHandler.GetTopicTypes)

		auth.POST("/notes", notesHandler.CreateNote)
		auth.PATCH("/notes/:id/delete", notesHandler.DeleteNote)
		auth.PATCH("/notes/:id/restore", notesHandler.RestoreNote)
		auth.GET("/notes/:id", notesHandler.GetNote)
		auth.GET("/notes", notesHandler.GetNotes)
		auth.GET("/tags/autocomplete", notesHandler.GetTagAutocomplete)

		auth.POST("/charts", chartsHandler.CreateChart)
		auth.PATCH("/charts/:id/delete", chartsHandler.DeleteChart)
		auth.PATCH("/charts/:id/restore", chartsHandler.RestoreChart)
		auth.PATCH("/charts/:id", chartsHandler.UpdateChart)
		auth.GET("/charts", chartsHandler.GetCharts)
		auth.GET("/chart-types", chartsHandler.GetChartTypes)
		auth.GET("/period-types", chartsHandler.GetPeriodTypes)

		auth.GET("/spaces/:spaceId/activity-types", activityTypesHandler.GetActivityTypesBySpace)
		auth.POST("/spaces/:spaceId/activity-types", activityTypesHandler.CreateActivityType)
		auth.PATCH("/spaces/:spaceId/activity-types/:typeId/delete", activityTypesHandler.DeleteActivityType)
		auth.PATCH("/spaces/:spaceId/activity-types/:typeId/restore", activityTypesHandler.RestoreActivityType)

		auth.POST("/activities", activitiesHandler.CreateActivity)
		auth.PATCH("/activities/:activityId/delete", activitiesHandler.DeleteActivity)
		auth.PATCH("/activities/:activityId/restore", activitiesHandler.RestoreActivity)
		auth.PATCH("/activities/:activityId", activitiesHandler.UpdateActivity)

		auth.GET("/activities", activitiesHandler.GetActivitiesAnalysis)

		auth.POST("/upload", attachmentsHandler.UploadFile)
		auth.GET("/files/:id", attachmentsHandler.GetFile)
	}

	r.Run(":8080")
}
