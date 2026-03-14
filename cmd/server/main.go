package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charan/url-shortener/internal/config"
	"github.com/charan/url-shortener/internal/database"
	"github.com/charan/url-shortener/internal/handler"
	"github.com/charan/url-shortener/internal/middleware"
	"github.com/charan/url-shortener/internal/repository/postgres"
	rediscache "github.com/charan/url-shortener/internal/repository/redis"
	"github.com/charan/url-shortener/internal/service"
	"github.com/charan/url-shortener/internal/worker"
	"github.com/charan/url-shortener/pkg/botdetect"
	"github.com/charan/url-shortener/pkg/geoip"
	"github.com/charan/url-shortener/pkg/useragent"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	ctx := context.Background()

	if cfg.RunMigrations {
		runMigrations(cfg)
	}

	pool, err := database.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	rdb, err := database.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		log.Printf("Warning: Redis unavailable at startup, continuing in degraded mode: %v", err)
		rdb = nil
	}
	if rdb != nil {
		defer rdb.Close()
	}
	middleware.SetRateLimitRedisClient(rdb)

	userRepo := postgres.NewUserRepo(pool)
	refreshSessionRepo := postgres.NewRefreshSessionRepo(pool)
	linkRepo := postgres.NewLinkRepo(pool)
	sourceLinkRepo := postgres.NewSourceLinkRepo(pool)
	groupRepo := postgres.NewGroupRepo(pool)
	clickEventRepo := postgres.NewClickEventRepo(pool)
	linkCache := rediscache.NewLinkCache(rdb, time.Duration(cfg.LinkCacheTTLSeconds)*time.Second)

	botDetector := botdetect.New()
	uaParser := useragent.New()
	geoResolver := geoip.New()

	clickWorker := worker.NewClickWorker(clickEventRepo, geoResolver, rdb)
	clickWorker.Start()
	defer clickWorker.Stop()

	authService := service.NewAuthService(userRepo, refreshSessionRepo, cfg.JWTSecret, cfg.GoogleClientID)
	linkService := service.NewLinkService(linkRepo, sourceLinkRepo, groupRepo, linkCache, cfg.BaseURL)
	redirectService := service.NewRedirectService(linkRepo, sourceLinkRepo, linkCache)
	analyticsService := service.NewAnalyticsService(clickEventRepo, linkRepo)

	authHandler := handler.NewAuthHandler(authService, cfg.RefreshCookieSecure, cfg.RefreshCookieDomain)
	linkHandler := handler.NewLinkHandler(linkService)
	redirectHandler := handler.NewRedirectHandler(redirectService, clickWorker, botDetector, uaParser)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)

	r := gin.Default()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("Failed to configure trusted proxies: %v", err)
	}
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLog())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/health/ready", func(c *gin.Context) {
		dbErr := pool.Ping(c.Request.Context())
		redisOK := true
		redisMode := "enabled"
		if rdb != nil {
			redisOK = rdb.Ping(c.Request.Context()).Err() == nil
		} else {
			redisOK = false
			redisMode = "degraded"
		}

		statusCode := http.StatusOK
		status := "ok"
		if dbErr != nil {
			statusCode = http.StatusServiceUnavailable
			status = "degraded"
		}

		if dbErr == nil && !redisOK {
			status = "degraded"
		}

		c.JSON(statusCode, gin.H{
			"status": status,
			"components": gin.H{
				"postgres": dbErr == nil,
				"redis":    redisOK,
				"mode":     redisMode,
			},
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.Request.URL.Path = "/health/ready"
		r.HandleContext(c)
	})

	auth := r.Group("/auth", middleware.RateLimit(cfg.AuthRateLimitPerMinute, time.Minute, "auth"))
	{
		// Authentication endpoints (public except /me).
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/google", authHandler.Google)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/me", middleware.Auth(cfg.JWTSecret), authHandler.Me)
	}

	api := r.Group(
		"/api",
		middleware.Auth(cfg.JWTSecret),
		middleware.RateLimit(cfg.APIRateLimitPerMinute, time.Minute, "api"),
	)
	{
		// Authenticated app APIs: links, groups, sources, and analytics.
		api.GET("/links", linkHandler.List)
		api.POST("/links", linkHandler.Create)
		api.GET("/links/:linkId", linkHandler.Get)
		api.PUT("/links/:linkId", linkHandler.Update)
		api.DELETE("/links/:linkId", linkHandler.Delete)

		api.GET("/links/:linkId/sources", linkHandler.ListSources)
		api.POST("/links/:linkId/sources", linkHandler.CreateSource)
		api.DELETE("/links/:linkId/sources/:sourceId", linkHandler.DeleteSource)
		api.POST("/sources/batch", linkHandler.BatchCreateSources)

		api.GET("/groups", linkHandler.ListGroups)
		api.POST("/groups", linkHandler.CreateGroup)
		api.PUT("/groups/:groupId", linkHandler.UpdateGroup)
		api.DELETE("/groups/:groupId", linkHandler.DeleteGroup)
		api.GET("/links/:linkId/groups", linkHandler.ListLinkGroups)
		api.POST("/groups/:groupId/links", linkHandler.AddLinkToGroup)
		api.DELETE("/groups/:groupId/links/:linkId", linkHandler.RemoveLinkFromGroup)

		api.GET("/links/:linkId/analytics", analyticsHandler.GetSummary)
		api.GET("/links/:linkId/analytics/clicks", analyticsHandler.GetClickTimeSeries)
		api.GET("/links/:linkId/analytics/sources", analyticsHandler.GetSourceBreakdown)
		api.GET("/links/:linkId/analytics/referrers", analyticsHandler.GetReferrerBreakdown)
		api.GET("/links/:linkId/analytics/locations", analyticsHandler.GetLocationBreakdown)
		api.GET("/links/:linkId/analytics/browsers", analyticsHandler.GetBrowserBreakdown)
		api.GET("/links/:linkId/analytics/recent", analyticsHandler.GetRecentActivity)
	}

	// Public short-link redirect endpoint.
	r.GET("/:hash", middleware.RateLimit(cfg.RedirectRateLimitPerMinute, time.Minute, "redirect"), redirectHandler.Redirect)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

func runMigrations(cfg *config.Config) {
	m, err := migrate.New("file://"+cfg.MigrationsDir, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations applied successfully")
}
