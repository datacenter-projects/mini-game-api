package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"app/pkg/configs"
	"app/pkg/middleware"
	"app/pkg/routes"
	"app/platform/database"
	"app/platform/logger"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gofiber/fiber/v2"
)

func main() {
	if err := configs.Load(); err != nil {
		log.Fatalf("config: %v", err)
	}
	logger.InitLogger(configs.Cfg.AppEnv == "local")
	defer logger.Logger.Sync() //nolint:errcheck

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		logger.SugarLogger.Fatalw("connect postgres", "error", err)
	}
	defer database.ClosePostgreSQL() //nolint:errcheck

	if configs.Cfg.MigrateOnStart {
		if err := database.Migrate(ctx, "up"); err != nil {
			logger.SugarLogger.Fatalw("migrate", "error", err) // migration พัง = ห้าม boot ต่อ
		}
	}

	if err := database.RedisConnection(ctx, configs.Cfg.Redis); err != nil {
		logger.SugarLogger.Fatalw("connect redis", "error", err)
	}
	defer database.CloseRedis() //nolint:errcheck

	app := fiber.New(configs.FiberConfig())
	middleware.FiberMiddleware(app)
	routes.SetupRoutes(app)

	go func() {
		logger.SugarLogger.Infow("server starting", "addr", configs.Cfg.ServerAddr, "env", configs.Cfg.AppEnv)
		if err := app.Listen(configs.Cfg.ServerAddr); err != nil {
			logger.SugarLogger.Errorw("server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.SugarLogger.Info("shutting down")
	if err := app.ShutdownWithTimeout(configs.Cfg.ShutdownTimeout); err != nil {
		logger.SugarLogger.Errorw("shutdown", "error", err)
	}
}
