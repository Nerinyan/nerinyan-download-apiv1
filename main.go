package main

import (
	"github.com/Nerinyan/nerinyan-download-apiv1/banchoCrawler"
	"github.com/Nerinyan/nerinyan-download-apiv1/config"
	"github.com/Nerinyan/nerinyan-download-apiv1/db/mariadb"
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"github.com/Nerinyan/nerinyan-download-apiv1/middlewareFunc"
	"github.com/Nerinyan/nerinyan-download-apiv1/route/common"
	"github.com/Nerinyan/nerinyan-download-apiv1/route/download"
	"github.com/Nerinyan/nerinyan-download-apiv1/webhook"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pterm/pterm"
	"net/http"
	"time"
)

func init() {
	ch := make(chan struct{})
	config.LoadConfig()
	mariadb.Connect()
	go banchoCrawler.LoadBancho(ch)
	_ = <-ch

	if config.Config.Debug {
	} else {
	}

}

func main() {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		logger.Errorf("%+v", err)
		_ = c.JSON(
			http.StatusInternalServerError, map[string]interface{}{
				"error":      err.Error(),
				"request_id": c.Response().Header().Get("X-Request-Id"),
				"time":       time.Now(),
			},
		)

	}
	e.Logger.SetOutput(logger.GetFileWriter())
	e.Renderer = &download.Renderer

	e.Pre(
		//필수 우선순
		middleware.Recover(),
		middleware.RequestID(),
		middleware.RemoveTrailingSlash(),
		middleware.Logger(),
		middlewareFunc.RequestConsolLogger(),
		middleware.RemoveTrailingSlash(),

		//1차 필터
		middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: []string{"*"}, AllowMethods: []string{echo.GET}}),

		//2차 필터

		//middleware.RateLimiterWithConfig(middleWareFunc.RateLimiterConfig),

	)

	// docs ============================================================================================================

	// 서버상태 체크용 ====================================================================================================
	e.GET("/health", common.Health)
	e.GET("/robots.txt", common.Robots)
	e.GET("/status", common.Status)

	// 맵 파일 다운로드 ===================================================================================================
	e.GET("/d/:setId", download.DownloadBeatmapSetV2, download.Embed)
	e.GET("/beatmapset/:setId", download.DownloadBeatmapSetV2)
	e.GET("/beatmap/:mapId", download.DownloadBeatmapSetV2)

	// 비트맵 BG  =========================================================================================================
	//e.GET("/bg/:id", download.BeatmapBG) // 사용 중지할 예정
	//e.GET("/bg/s/:sid", download.BeatmapBG)
	//e.GET("/bg/m/:mid", download.BeatmapBG)

	// 개발중 || 테스트중 ===================================================================================================

	// ====================================================================================================================
	pterm.Info.Println("ECHO STARTED AT", config.Config.Port)
	webhook.DiscordInfoStartUP()
	e.Logger.Fatal(e.Start(":" + config.Config.Port))

}
