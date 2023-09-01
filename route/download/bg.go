package download

import (
	"errors"
	"fmt"
	"github.com/Nerinyan/nerinyan-download-apiv1/db/mariadb"
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"github.com/Nerinyan/nerinyan-download-apiv1/utils"
	"github.com/labstack/echo/v4"
	"io"
	"os"
	"time"
)

func (bgReq) TableName() string {
	return _TB_BEATMAPSET
}

type bgReq struct {
	Id           int `param:"id"`
	Sid          int `param:"setId"`
	Mid          int `param:"mapId"`
	IsMap        bool
	IsSet        bool
	BeatmapId    int
	BeatmapsetId int
	Version      string
	LastUpdated  time.Time
}

func (bgSet) TableName() string {
	return _TB_BEATMAPSET
}

type bgSet struct {
	BeatmapsetId int       `gorm:"column:BEATMAPSET_ID"`
	LastUpdated  time.Time `gorm:"column:LAST_UPDATED"`
	Map          *bgMap    `gorm:"foreignKey:BEATMAPSET_ID;references:BEATMAPSET_ID"`
}

func (bgMap) TableName() string {
	return _TB_BEATMAP
}

type bgMap struct {
	BeatmapId    int    `gorm:"column:BEATMAP_ID"`
	BeatmapsetId int    `gorm:"column:BEATMAPSET_ID"`
	Version      string `gorm:"column:VERSION"`
	Set          *bgSet `gorm:"foreignKey:BEATMAPSET_ID;references:BEATMAPSET_ID"`
}

func BeatmapBG(c echo.Context) (err error) {

	var req bgReq

	err = c.Bind(&req)
	if err != nil {
		return
	}
	if req.Id == 0 && req.Sid == 0 && req.Mid == 0 {
		return errors.New("id cannot '0'")
	}
	for {
		// 맵셋
		if req.Id != 0 || req.Sid != 0 { // 셋 id
			bset := bgSet{BeatmapsetId: req.Id + req.Sid}
			err = mariadb.Mariadb.Find(&bset).Preload("Map").Error
			if err == nil {
				req.IsSet = true
				req.BeatmapsetId = bset.BeatmapsetId
				req.LastUpdated = bset.LastUpdated
				break
			} else {
				logger.Error(err)
			}
		}

		// id 검색(set error)인경우 || 맵 검색인경우
		if req.Id != 0 || req.Mid != 0 { // 맵 id
			bmap := bgMap{BeatmapId: req.Id + req.Mid}
			err = mariadb.Mariadb.Find(&bmap).Preload("Map").Error
			if err == nil {
				req.IsSet = true
				req.BeatmapsetId = bmap.BeatmapsetId
				req.Version = bmap.Version
				if bmap.Set != nil {
					req.LastUpdated = bmap.Set.LastUpdated
				}

				break
			} else {
				logger.Error(err)
			}
		}

		// 여기까지 왔다는것은 일치하는맵을 찾지 못했다는것임
		return errors.New("set id & map id not found")
	}
	// 맵이 데이터베이스에 존재하는경우
	// 다운로드가 비활성화 되었더라도 mp3 만 제거되는것이라 배경에는 영향 없음.
	//data, err := getBeatmapData4BG(req.BeatmapsetId, req.LastUpdated) // 저장 다 하고 리턴하는것임
	//if err != nil {
	//    logger.Error(err)
	//    return
	//}

	//zip 에서 추출

	return
}

func getBeatmapData4BG(sid int, ttl time.Time) (data []byte, err error) {

	if file, _ := os.Open(getSourceOszPath(sid)); file != nil { // 원본이 있는지 확인함
		defer file.Close()
		if stat, _ := file.Stat(); stat != nil && !stat.IsDir() && stat.ModTime().After(ttl) { // 만료되지 않은 파일인경우
			logger.Infof("return cached [%s] file. modify at '%s'", getSourceOszPath(sid), stat.ModTime().Format(time.RFC3339))
			return io.ReadAll(file)
		}
	}
	//===================================
	// 여기서부터는 최초다운로드이기때문에 저장이 필요함
	var reader io.ReadCloser
	var length int64

	defer func() {
		if err != nil && len(data) > 0 {
			if e := utils.Save2File(data, getSourceOszPath(sid)); e != nil { // 원본 osz 저장
				logger.Errorf("failed to save [%s] error: %s", getSourceOszPath(sid), e)
			}
		}
	}()

	// TODO 시간날때 중복코드 제거 해야함
	if !isLimitedDownload() {
		reader, length, err = downloadFromBancho2(sid)
		if err == nil {
			logger.Info("use bancho datasource")
			defer reader.Close()
			data, err = io.ReadAll(reader)
			if err != nil {
				logger.Error("failed bancho datasource ", err)
			}
			if int64(len(data)) != length {
				err = fmt.Errorf("contentLength: %d, RX bytes: %d download failed", length, int64(len(data)))
			}
			return
		} else {
			logger.Error("failed bancho datasource ", err)
		}
	}

	logger.Info("use beatconnect datasource")
	reader, length, err = downloadFromBeatconnect2(sid)
	if err == nil {
		logger.Info("use beatconnect datasource")
		defer reader.Close()
		data, err = io.ReadAll(reader)
		if err != nil {
			logger.Error("failed beatconnect datasource ", err)
		}
		if int64(len(data)) != length {
			err = fmt.Errorf("contentLength: %d, RX bytes: %d download failed", length, int64(len(data)))
		}
		return
	} else {
		logger.Error("failed beatconnect datasource ", err)
	}

	return

}
