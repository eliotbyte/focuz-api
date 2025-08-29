package main

import (
	"database/sql"
	"focuz-api/handlers"
	"focuz-api/initializers"
	"focuz-api/middleware"
	"focuz-api/pkg/notify"
	"focuz-api/repository"
	"focuz-api/websocket"
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
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 characters")
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
	notesRepo := repository.NewNotesRepository(db)
	rolesRepo := repository.NewRolesRepository(db)
	activityTypesRepo := repository.NewActivityTypesRepository(db)
	activitiesRepo := repository.NewActivitiesRepository(db)
	attachmentsRepo := repository.NewAttachmentsRepository(db)
	chartsRepo := repository.NewChartsRepository(db)
	notificationsRepo := repository.NewNotificationsRepository(db)
	filtersRepo := repository.NewFiltersRepository(db)

	// New repos for sync and tags
	syncRepo := repository.NewSyncRepository(db)
	tagsRepo := repository.NewTagsRepository(db)

	r := gin.New()
	// Structured request ID and JSON access logs
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.LoggerMiddleware())
	// Panic recovery
	r.Use(gin.Recovery())

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
		_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}

	r.Use(middleware.CORSMiddleware())
	// Apply rate limiting globally after CORS but before routes
	r.Use(middleware.RateLimitMiddleware())

	// Initialize WebSocket hub and notifier
	hub := websocket.NewHub()
	notifier := &notify.WSNotifier{Hub: hub}

	// Public endpoints
	r.GET("/health", handlers.HealthCheck)

	// Auth-protected WebSocket
	auth := r.Group("/", handlers.AuthMiddleware(jwtSecret))
	{
		auth.GET("/ws", websocket.ServeWS(hub))
	}

	// Handlers
	notesHandler := handlers.NewNotesHandler(notesRepo, spacesRepo)
	spacesHandler := handlers.NewSpacesHandler(spacesRepo, rolesRepo).WithNotifier(notifier).WithNotificationsRepo(notificationsRepo)
	activityTypesHandler := handlers.NewActivityTypesHandler(activityTypesRepo, spacesRepo)
	activitiesHandler := handlers.NewActivitiesHandler(
		activitiesRepo,
		spacesRepo,
		notesRepo,
		activityTypesRepo,
	)
	attachmentsHandler := handlers.NewAttachmentsHandler(attachmentsRepo, notesRepo, spacesRepo)
	chartsHandler := handlers.NewChartsHandler(chartsRepo, spacesRepo, activityTypesRepo, notesRepo)
	notificationsHandler := handlers.NewNotificationsHandler(notificationsRepo)
	filtersHandler := handlers.NewFiltersHandler(filtersRepo, spacesRepo)
	syncHandler := handlers.NewSyncHandler(syncRepo, spacesRepo, tagsRepo, filtersRepo)

	// Set Gin to release mode in production
	if os.Getenv("GIN_MODE") == "release" || strings.ToLower(os.Getenv("APP_ENV")) == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Public endpoints with stricter auth rate limit
	authPublic := r.Group("/", middleware.RateLimitAuthMiddleware())
	authPublic.POST("/register", notesHandler.Register)
	authPublic.POST("/login", func(c *gin.Context) {
		c.Set("jwtSecret", jwtSecret)
		notesHandler.Login(c)
	})

	auth = r.Group("/", handlers.AuthMiddleware(jwtSecret))
	{
		auth.GET("/spaces", spacesHandler.GetAccessibleSpaces)
		auth.DELETE("/spaces/:spaceId/users/:userId", spacesHandler.RemoveUser)
		auth.GET("/spaces/:spaceId/users", spacesHandler.GetUsersInSpace)
		auth.POST("/spaces", spacesHandler.CreateSpace)
		auth.PATCH("/spaces/:spaceId", spacesHandler.UpdateSpace)
		auth.PATCH("/spaces/:spaceId/delete", spacesHandler.DeleteSpace)
		auth.PATCH("/spaces/:spaceId/restore", spacesHandler.RestoreSpace)
		auth.POST("/spaces/:spaceId/invite", spacesHandler.InviteUser)
		auth.POST("/spaces/:spaceId/invitations/accept", spacesHandler.AcceptInvitation)
		auth.POST("/spaces/:spaceId/invitations/decline", spacesHandler.DeclineInvitation)

		// notes (legacy, kept for backward compatibility during migration)
		auth.POST("/notes", notesHandler.CreateNote)
		auth.PATCH("/notes/:id/delete", notesHandler.DeleteNote)
		auth.PATCH("/notes/:id/restore", notesHandler.RestoreNote)
		auth.GET("/notes/:id", notesHandler.GetNote)
		auth.GET("/notes", notesHandler.GetNotes)
		auth.GET("/tags/autocomplete", notesHandler.GetTagAutocomplete)

		// charts
		auth.POST("/charts", chartsHandler.CreateChart)
		auth.PATCH("/charts/:id/delete", chartsHandler.DeleteChart)
		auth.PATCH("/charts/:id/restore", chartsHandler.RestoreChart)
		auth.PATCH("/charts/:id", chartsHandler.UpdateChart)
		auth.GET("/charts", chartsHandler.GetCharts)
		auth.GET("/chart-types", chartsHandler.GetChartTypes)
		auth.GET("/period-types", chartsHandler.GetPeriodTypes)
		auth.GET("/charts/:id/data", chartsHandler.GetChartData)

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
		auth.GET("/notifications/unread", notificationsHandler.ListUnread)
		auth.POST("/notifications/mark-read", notificationsHandler.MarkRead)

		// filters
		auth.POST("/filters", filtersHandler.Create)
		auth.GET("/filters", filtersHandler.List)
		auth.PATCH("/filters/:id", filtersHandler.Update)
		auth.PATCH("/filters/:id/delete", filtersHandler.Delete)
		auth.PATCH("/filters/:id/restore", filtersHandler.Restore)

		// New sync and utility endpoints
		auth.GET("/sync", syncHandler.Pull)
		auth.POST("/sync", syncHandler.Push)
		auth.GET("/spaces/:spaceId/tags", syncHandler.GetTagsBySpace)
		auth.GET("/spaces/:spaceId/filters", syncHandler.GetFiltersBySpace)
	}

	r.Run(":8080")
}
