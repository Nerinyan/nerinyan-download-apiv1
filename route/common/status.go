package common

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"runtime"
)

func Status(c echo.Context) error {
	return c.JSON(
		http.StatusOK, map[string]interface{}{
			"cpuThreadCount":        runtime.NumCPU(),
			"runningGoroutineCount": runtime.NumGoroutine(),
			//"apiCount":              *banchoCrawler.ApiCount,
		},
	)
}
