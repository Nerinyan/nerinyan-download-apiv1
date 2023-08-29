package middlewareFunc

import (
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"github.com/labstack/echo/v4"
)

func RequestConsolLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if path != "/health" {
				logger.Infof(
					// 2023-05-13T00:01:02Z | GET    | 123.123.123.123 | api.nerinyan.moe | /search
					"| %6s | %15s | %-30s | %-s",
					c.Request().Method, c.RealIP(), c.Request().Host, c.Request().URL.Path,
				)
			}
			return next(c)
		}
	}
}
