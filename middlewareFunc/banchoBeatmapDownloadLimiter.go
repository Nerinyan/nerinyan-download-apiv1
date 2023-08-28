package middlewareFunc

import (
	"github.com/labstack/echo/v4"
	"time"
)

var requestTime map[string][]int64

func init() {
	go func() {
		for {
			exp := time.Now().UnixMilli() - (time.Minute.Milliseconds() * 10)
			for k, v := range requestTime {
				vl := len(v)
				var t []int64
				for i := 0; i < vl; i++ {
					if v[i] > exp {
						t = append(t, v[i])
					}
				}
				requestTime[k] = t
			}
			time.Sleep(time.Second)
		}
	}()
}

func BanchoBeatmapDownloadLimiter(h echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if len(requestTime[c.Request().Host]) < 150 {
			return h(c)
		}
		return h(c)
	}
}
