package main

import (
	"context"
	"log"
	"os"

	"voltana-api/internal/handler"
	"voltana-api/internal/mailer"
	"voltana-api/internal/middleware"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	isProd := os.Getenv("APP_ENV") == "production"
	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	// ── Database ──────────────────────────────────────────────────────────────
	dbURL := mustEnv("DATABASE_URL")
	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}
	log.Println("postgres: connected")

	// ── Redis ─────────────────────────────────────────────────────────────────
	redisOpt, err := redis.ParseURL(mustEnv("REDIS_URL"))
	if err != nil {
		log.Fatalf("parse REDIS_URL: %v", err)
	}
	rdb := redis.NewClient(redisOpt)
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("ping redis: %v", err)
	}
	log.Println("redis: connected")

	// ── Dependencies ──────────────────────────────────────────────────────────
	userRepo     := repository.NewUserRepository(db)
	verifRepo    := repository.NewVerificationTokenRepository(db)
	tokenStore   := repository.NewRedisTokenStore(rdb)
	carRepo      := repository.NewCarRepository(db)
	evModelRepo  := repository.NewEVModelRepository(db)
	chargingRepo := repository.NewChargingRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	batteryRepo  := repository.NewBatteryRepository(db)
	stationRepo  := repository.NewStationRepository(db)

	// Mailer: real SMTP when SMTP_HOST is set, otherwise a dev log mailer that
	// never prints the verification link/token.
	var mail service.Mailer
	if host := os.Getenv("SMTP_HOST"); host != "" {
		mail = mailer.NewSMTP(host, getenv("SMTP_PORT", "587"), os.Getenv("SMTP_USER"),
			os.Getenv("SMTP_PASSWORD"), getenv("SMTP_FROM", "noreply@voltana.app"))
		log.Println("mailer: SMTP configured")
	} else {
		mail = mailer.LogMailer{}
		log.Println("mailer: SMTP not configured — using dev log mailer")
	}

	authSvc     := service.NewAuthService(userRepo, verifRepo, tokenStore, mail, os.Getenv("APP_URL"), mustEnv("JWT_SECRET"))
	carSvc      := service.NewCarService(carRepo)
	evModelSvc  := service.NewEVModelService(evModelRepo)
	chargingSvc := service.NewChargingService(chargingRepo, carRepo, settingsRepo)
	settingsSvc := service.NewSettingsService(settingsRepo, carRepo)
	analyticsSvc := service.NewAnalyticsService(carRepo, evModelRepo, chargingRepo, batteryRepo, tokenStore)
	stationSvc  := service.NewStationService(stationRepo)

	// Recompute battery SOH off the request path whenever charging history changes.
	chargingSvc.SetHealthRecomputer(analyticsSvc)

	authH      := handler.NewAuthHandler(authSvc, isProd)
	carH       := handler.NewCarHandler(carSvc)
	evModelH   := handler.NewEVModelHandler(evModelSvc)
	chargingH  := handler.NewChargingHandler(chargingSvc)
	settingsH  := handler.NewSettingsHandler(settingsSvc)
	analyticsH := handler.NewAnalyticsHandler(analyticsSvc)
	stationH   := handler.NewStationHandler(stationSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	r := gin.New()

	// Trust only the Docker bridge network (where nginx sits). Without this Gin
	// trusts all proxies and would return the client-controlled left-most
	// X-Forwarded-For entry — letting an attacker rotate the header to bypass
	// the per-IP login rate limit.
	if err := r.SetTrustedProxies([]string{"172.16.0.0/12"}); err != nil {
		log.Fatalf("set trusted proxies: %v", err)
	}
	// Derive the client IP from X-Real-IP, which nginx sets from $remote_addr and
	// overwrites on every request — clients cannot forge it. X-Forwarded-For is
	// client-appendable, so it must not back the rate-limit key.
	r.RemoteIPHeaders = []string{"X-Real-IP"}

	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", handler.Health)

	auth := r.Group("/auth")
	{
		auth.POST("/register",            authH.Register)
		auth.POST("/login",               authH.Login)
		auth.POST("/refresh",             authH.Refresh)
		auth.POST("/logout",              authH.Logout)
		auth.POST("/verify-email",        authH.VerifyEmail)
		auth.POST("/resend-verification", authH.ResendVerification)
	}

	// /v1 — all routes require a valid Bearer access token (Auth middleware).
	v1 := r.Group("/v1", middleware.Auth(authSvc))
	{
		v1.GET("/cars", carH.List)
		v1.POST("/cars", carH.Create)
		v1.GET("/cars/:id", carH.Get)
		v1.PUT("/cars/:id", carH.Update)
		v1.DELETE("/cars/:id", carH.Delete)

		v1.GET("/ev-models", evModelH.List)
		v1.GET("/ev-models/:id", evModelH.Get)

		v1.GET("/charging-sessions", chargingH.List)
		v1.POST("/charging-sessions", chargingH.Create)
		v1.GET("/charging-sessions/:id", chargingH.Get)
		v1.PUT("/charging-sessions/:id", chargingH.Update)
		v1.DELETE("/charging-sessions/:id", chargingH.Delete)

		v1.GET("/settings", settingsH.Get)
		v1.PUT("/settings", settingsH.Update)

		v1.GET("/analytics/dashboard", analyticsH.Dashboard)
		v1.GET("/analytics/battery/:car_id", analyticsH.Battery)
		v1.GET("/analytics/battery/:car_id/history", analyticsH.BatteryHistory)
		v1.GET("/analytics/recommendations/:car_id", analyticsH.Recommendations)

		// Charging stations: reads open to any authed user; writes admin-only.
		v1.GET("/stations", stationH.List)
		v1.GET("/stations/:id", stationH.Get)
		v1.POST("/stations", middleware.AdminOnly(authSvc), stationH.Create)
		v1.PUT("/stations/:id", middleware.AdminOnly(authSvc), stationH.Update)
		v1.DELETE("/stations/:id", middleware.AdminOnly(authSvc), stationH.Delete)
	}

	// ── Start ─────────────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	log.Printf("server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
