package main

import (
	"context"
	"log"
	"os"

	"voltana-api/internal/bot"
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

	// ── Mailer ────────────────────────────────────────────────────────────────
	var mail service.Mailer
	if host := os.Getenv("SMTP_HOST"); host != "" {
		mail = mailer.NewSMTP(host, getenv("SMTP_PORT", "587"), os.Getenv("SMTP_USER"),
			os.Getenv("SMTP_PASSWORD"), getenv("SMTP_FROM", "noreply@voltana.app"))
		log.Println("mailer: SMTP configured")
	} else {
		mail = mailer.LogMailer{}
		log.Println("mailer: SMTP not configured — using dev log mailer")
	}

	// ── Auth service ──────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(userRepo, verifRepo, tokenStore, mail, os.Getenv("APP_URL"), mustEnv("JWT_SECRET"))

	// ── Bot OTP senders (TASK-0017) ───────────────────────────────────────────
	// Senders are wired only when the corresponding bot token is present.
	// With no tokens configured, LogOTPSender is used so QA can test the OTP
	// path without a real bot (code appears in the server log).
	baleToken   := os.Getenv("BALE_BOT_TOKEN")
	tgToken     := os.Getenv("TELEGRAM_BOT_TOKEN")
	baleUser    := os.Getenv("BALE_BOT_USERNAME")
	tgUser      := os.Getenv("TELEGRAM_BOT_USERNAME")

	var baleSender, tgSender service.OTPSender
	switch {
	case baleToken != "":
		baleSender = bot.NewBaleSender(baleToken)
		log.Println("bot: Bale sender configured")
	default:
		baleSender = bot.LogOTPSender{}
		log.Println("bot: BALE_BOT_TOKEN not set — using LogOTPSender")
	}
	if tgToken != "" {
		tgSender = bot.NewTelegramSender(tgToken)
		log.Println("bot: Telegram sender configured")
	}

	authSvc.SetBotSenders(baleSender, tgSender, baleUser, tgUser)

	// ── Long-poll workers ─────────────────────────────────────────────────────
	// One poller per configured real bot token. Pollers run in-process goroutines
	// (outbound-only HTTPS — no public webhook needed, safe behind NAT / WSL2).
	ctx := context.Background()
	if baleToken != "" {
		poller := bot.NewPoller("https://api.bale.ai/bot"+baleToken, "bale", authSvc)
		go poller.Run(ctx)
	}
	if tgToken != "" {
		tgBaseURL := "https://api.telegram.org/bot" + tgToken
		if err := bot.ProbeBot(tgBaseURL); err != nil {
			log.Printf("bot: Telegram unreachable (%v) — poller not started (may be filtered by ISP)", err)
		} else {
			poller := bot.NewPoller(tgBaseURL, "telegram", authSvc)
			go poller.Run(ctx)
		}
	}

	// ── Other services ────────────────────────────────────────────────────────
	carSvc      := service.NewCarService(carRepo)
	evModelSvc  := service.NewEVModelService(evModelRepo)
	chargingSvc := service.NewChargingService(chargingRepo, carRepo, settingsRepo)
	settingsSvc := service.NewSettingsService(settingsRepo, carRepo)
	analyticsSvc := service.NewAnalyticsService(carRepo, evModelRepo, chargingRepo, batteryRepo, tokenStore)
	stationSvc  := service.NewStationService(stationRepo)

	chargingSvc.SetHealthRecomputer(analyticsSvc)

	authH      := handler.NewAuthHandler(authSvc, isProd)
	accountH   := handler.NewAccountHandler(authSvc)
	carH       := handler.NewCarHandler(carSvc)
	evModelH   := handler.NewEVModelHandler(evModelSvc)
	chargingH  := handler.NewChargingHandler(chargingSvc)
	settingsH  := handler.NewSettingsHandler(settingsSvc)
	analyticsH := handler.NewAnalyticsHandler(analyticsSvc)
	stationH   := handler.NewStationHandler(stationSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	r := gin.New()

	if err := r.SetTrustedProxies([]string{"172.16.0.0/12"}); err != nil {
		log.Fatalf("set trusted proxies: %v", err)
	}
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
		auth.POST("/otp/request",         authH.OTPRequest)
		auth.POST("/otp/verify",          authH.OTPVerify)
		auth.POST("/otp/register",        authH.OTPRegister)
	}

	v1 := r.Group("/v1", middleware.Auth(authSvc))
	{
		v1.GET("/me", authH.Me)

		v1.POST("/account/bot-link", accountH.BotLink)

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
