package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var Logger *slog.Logger

func InitLogger() {
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}

	// Create JSON handler for production, text handler for development
	var handler slog.Handler
	handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

func StructuredLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			// makes log less noisy by skipping static assets requests
			return path == "/static" || len(path) > 8 && path[:8] == "/static/"
		},
		LogMethod:   true,
		LogURIPath:  true,
		LogStatus:   true,
		LogRemoteIP: true,
		LogError:    true,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				Logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("method", v.Method),
					slog.String("path", v.URIPath),
					slog.Int("status", v.Status),
					slog.String("remoteIP", v.RemoteIP),
				)
			} else {
				Logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("method", v.Method),
					slog.String("path", v.URIPath),
					slog.Int("status", v.Status),
					slog.String("remoteIP", v.RemoteIP),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	})

}
