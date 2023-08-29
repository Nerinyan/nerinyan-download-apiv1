package main

import (
	"github.com/Nerinyan/nerinyan-download-apiv1/banchoCrawler"
	"github.com/Nerinyan/nerinyan-download-apiv1/config"
	"github.com/Nerinyan/nerinyan-download-apiv1/db/mariadb"
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"github.com/Nerinyan/nerinyan-download-apiv1/middlewareFunc"
	"github.com/Nerinyan/nerinyan-download-apiv1/route/common"
	"github.com/Nerinyan/nerinyan-download-apiv1/route/download"
	"github.com/Nerinyan/nerinyan-download-apiv1/src"
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
	src.StartIndex()
	//middlewareFunc.StartHandler()
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
				"error":      err,
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
		middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: []string{"*"}, AllowMethods: []string{echo.GET, echo.HEAD, echo.POST}}),

		//2차 필터

		//middleware.RateLimiterWithConfig(middleWareFunc.RateLimiterConfig),

	)

	// docs ============================================================================================================

	// 서버상태 체크용 ====================================================================================================
	e.GET("/health", common.Health)
	e.GET("/robots.txt", common.Robots)
	e.GET("/status", common.Status)

	// 맵 파일 다운로드 ===================================================================================================
	e.GET("/d/:setId", download.DownloadBeatmapSetV2)
	//e.GET("/d/:setId", download.DownloadBeatmapSet, download.Embed)
	//e.GET("/beatmap/:mapId", download.DownloadBeatmapSet)
	//e.GET("/beatmapset/:setId", download.DownloadBeatmapSet)
	//TODO 맵아이디, 맵셋아이디 지원

	// 비트맵 BG  =========================================================================================================
	//e.GET(
	//    "/bg/:setId", func(c echo.Context) error {
	//        redirectUrl := "https://subapi.nerinyan.moe/bg/" + c.Param("setId")
	//        return c.Redirect(http.StatusPermanentRedirect, redirectUrl)
	//    },
	//)

	// 개발중 || 테스트중 ===================================================================================================

	// ====================================================================================================================
	pterm.Info.Println("ECHO STARTED AT", config.Config.Port)
	webhook.DiscordInfoStartUP()
	e.Logger.Fatal(e.Start(":" + config.Config.Port))

}
