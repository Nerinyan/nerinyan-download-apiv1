package download

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"github.com/Nerinyan/nerinyan-download-apiv1/config"
	"github.com/Nerinyan/nerinyan-download-apiv1/db/mariadb"
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"github.com/Nerinyan/nerinyan-download-apiv1/utils"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type oszReq struct {
	Sid             int       `param:"setId"`
	Mid             int       `param:"mapId"`
	Novideo         bool      `query:"noVideo"`
	Nobackground    bool      `query:"noBg"`
	NoHitsound      bool      `query:"noHitsound"`
	NoStoryboard    bool      `query:"noStoryboard"`
	Nv              bool      `query:"nv"`
	Nh              bool      `query:"nh"`
	Nb              bool      `query:"nb"`
	Ns              bool      `query:"nsb"`
	BeatmapsetId    int       `gorm:"column:BEATMAPSET_ID"`
	Artist          string    `gorm:"column:ARTIST"`
	Creator         string    `gorm:"column:CREATOR"`
	Title           string    `gorm:"column:TITLE"`
	LastUpdated     time.Time `gorm:"column:LAST_UPDATED"`
	DownloadDisable bool      `gorm:"column:AVAILABILITY_DOWNLOAD_DISABLED"`
}

func (oszReq) TableName() string {
	return _TB_BEATMAPSET
}
func (v *oszReq) isModify() bool {
	return v.Nv || v.Novideo || v.Nb || v.Nobackground || v.Nh || v.NoHitsound || v.Ns || v.NoStoryboard
}

func (v *oszReq) getSourceFileName() (path string) {
	path = fmt.Sprintf("%s/%d/%d", config.Config.TargetDir, v.BeatmapsetId, v.BeatmapsetId)
	path += "." + _SERVER_OSZ_EXT
	return
}

func (v *oszReq) getOptionFileName() (path string) {
	path = fmt.Sprintf("%s/%d/%d", config.Config.TargetDir, v.BeatmapsetId, v.BeatmapsetId)
	var args []string

	if v.Nv || v.Novideo {
		args = append(args, "nv")
	}
	if v.Nb || v.Nobackground {
		args = append(args, "nb")
	}
	if v.Nh || v.NoHitsound {
		args = append(args, "nh")
	}
	if v.Ns || v.NoStoryboard {
		args = append(args, "ns")
	}
	if len(args) > 0 {
		path += "_" + strings.Join(args, "_")
	}
	path += "." + _SERVER_OSZ_EXT
	return

}
func (v *oszReq) getFilterRegex() (regexps []*regexp.Regexp) {
	if v.Nv || v.Novideo {
		regexps = append(regexps, _REGEXP_NV)
	}
	if v.Nb || v.Nobackground {
		regexps = append(regexps, _REGEXP_NB)
	}
	if v.Nh || v.NoHitsound {
		regexps = append(regexps, _REGEXP_NH)
	}
	if v.Ns || v.NoStoryboard {
		regexps = append(regexps, _REGEXP_NS)
	}
	return

}
func (v *oszReq) GetClientFilename() string {
	return _REGEXP_FN_NOT_ALLOW.ReplaceAllString(fmt.Sprintf("%d %s - %s.osz", v.BeatmapsetId, v.Artist, v.Title), "_")
}

func DownloadBeatmapSetV2(c echo.Context) (err error) {

	var req oszReq
	err = c.Bind(&req)
	if err != nil {
		logger.Error(err)
		return
	}
	if req.Sid != 0 {
		err = mariadb.Mariadb.Model(&oszReq{}).Where(&oszReq{BeatmapsetId: req.Sid}).Find(&req).Error
	} else if req.Mid != 0 {
		err = mariadb.Mariadb.Model(&oszReq{}).Where("BEATMAPSET_ID = (SELECT BEATMAPSET_ID FROM BEATMAP WHERE BEATMAP_ID = ?)", req.Mid).Find(&req).Error
	}

	if req.BeatmapsetId == 0 {
		err = errors.New("set id & map id not found")
	}

	if err != nil {
		logger.Error(err)
		return
	}
	if req.DownloadDisable {
		logger.Errorf("beatmapset %d download disabled", req.BeatmapsetId)
		return fmt.Errorf("beatmapset %d download disabled", req.BeatmapsetId)
	}

	// 유효한 요청인지 체크
	//=====================================================================================================================
	//=====================================================================================================================
	//=====================================================================================================================
	// 옵션파싱, 캐싱여부 확인, 다운로드 스트림 생성
	reader, length, cached, save, err := getBeatmapData(req)
	if err != nil {
		logger.Error(err)
		return
	}
	defer reader.Close()

	//=============================================
	// 클라이언트 응답.
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, req.GetClientFilename()))
	c.Response().Header().Set(echo.HeaderContentType, "application/x-osu-beatmap-archive")

	//=============================================
	// 파일이 캐시된경우
	if cached {
		c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(length, 10))
		return c.Stream(http.StatusOK, "application/x-osu-beatmap-archive", reader)
	}

	//==========================================================================================
	// 파일이 만료되었거나 서버에 없어서 다운로드 받는경우
	//==========================================================================================
	//=============================================
	// 재작업 없이 리턴하는경우

	if !req.isModify() {
		c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(length, 10))
		oszBuf, e := utils.CaptureRW(reader, c.Response().Writer)
		c.Response().Flush()
		if e != nil {
			logger.Error(e)
			return e
		}

		if int64(oszBuf.Len()) != length { // 바이트 길이가 맞지 않는경우
			err = fmt.Errorf("contentLength: %d, RX bytes: %d download failed", length, int64(oszBuf.Len()))
			logger.Error(err)
			return
		}

		// 여기서 발생하는 에러는 서버의 문제임으로 클라이언트에 리턴하지 않는다
		if save {
			if e = utils.Save2File(oszBuf.Bytes(), req.getSourceFileName()); e != nil { // 원본 osz 저장
				logger.Errorf("failed to save [%s] error: %s", req.getSourceFileName(), e)
			}
		}

		return

	}

	//=============================================
	// 재작업이 필요한경우

	if req.isModify() {
		logger.Infof("nv: %t, nb: %t, nh: %t, ns: %t", req.Nv, req.Nb, req.Nh, req.Ns)
		data, e := io.ReadAll(reader)
		if e != nil {
			logger.Error(e)
			return e
		}
		if save {
			if e = utils.Save2File(data, req.getSourceFileName()); e != nil { // 원본 osz 저장
				logger.Errorf("failed to save [%s] error: %s", req.getSourceFileName(), e)
			}
		}

		rd, e := rebuildOsz(data, req.getFilterRegex())
		if e != nil {
			logger.Error(e)
			return e
		}

		c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(int64(rd.Len()), 10))
		oszBuf, e := utils.CaptureRW(&rd, c.Response().Writer)
		c.Response().Flush()
		if e != nil {
			logger.Error(e)
			return e
		}

		// 여기서 발생하는 에러는 서버의 문제임으로 클라이언트에 리턴하지 않는다
		if e = utils.Save2File(oszBuf.Bytes(), req.getOptionFileName()); e != nil { // 수정한 osz 저장
			logger.Errorf("failed to save [%s] error: %s", req.getOptionFileName(), e)
			return
		}
	}

	return

}

// 에러가 발생한경우 스트림을 닫고 에러를 리턴하고
// 에러가 없는경우 열려있는 스트림을 리턴함
func getBeatmapData(req oszReq) (reader io.ReadCloser, length int64, cached, save bool, err error) {
	if file, _ := os.Open(req.getOptionFileName()); file != nil { // 정확히 일치하는게 있는지 확인함
		if stat, _ := file.Stat(); stat != nil && !stat.IsDir() && stat.ModTime().After(req.LastUpdated) { // 만료되지 않은 파일인경우
			reader = file
			length = stat.Size()
			cached = true
			logger.Infof("return cached [%s] file. modify at '%s'", req.getOptionFileName(), stat.ModTime().Format(time.RFC3339))
			return
		} else {
			_ = file.Close()
		}
	}

	if file, _ := os.Open(req.getSourceFileName()); file != nil { // 원본이 있는지 확인함
		if stat, _ := file.Stat(); stat != nil && !stat.IsDir() && stat.ModTime().After(req.LastUpdated) { // 만료되지 않은 파일인경우
			reader = file
			length = stat.Size()
			logger.Infof("return cached [%s] file. modify at '%s'", req.getSourceFileName(), stat.ModTime().Format(time.RFC3339))
			return
		} else {
			_ = file.Close()
		}
	}
	//===================================
	// 여기서부터는 최초다운로드이기때문에 저장이 필요함
	save = true
	if !isLimitedDownload() {
		reader, length, err = downloadFromBancho2(req.BeatmapsetId)
		if err == nil {
			logger.Info("use bancho datasource")
			return
		} else {
			logger.Error("failed bancho datasource ", err)
		}
	}
	logger.Info("use beatconnect datasource")
	reader, length, err = downloadFromBeatconnect2(req.BeatmapsetId)
	if err != nil {
		logger.Error("failed beatconnect datasource ", err)
	}
	return

}

func rebuildOsz(data []byte, notInRegexp []*regexp.Regexp) (res bytes.Buffer, err error) {
	// ./beatmaps/{sid}/{sid}.osz
	// ./beatmaps/{sid}/{sid}_nv.osz
	// ./beatmaps/{sid}/{sid}_nb.osz
	// ./beatmaps/{sid}/{sid}_nh.osz
	// ./beatmaps/{sid}/{sid}_ns.osz
	// ./beatmaps/{sid}/{sid}_nv_nb.osz
	// ./beatmaps/{sid}/{sid}_nv_nh.osz
	// ./beatmaps/{sid}/{sid}_nv_ns.osz
	// ./beatmaps/{sid}/{sid}_nb_nh.osz
	// ./beatmaps/{sid}/{sid}_nb_ns.osz
	// ./beatmaps/{sid}/{sid}_nh_ns.osz
	// ./beatmaps/{sid}/{sid}_nv_nb_nh.osz
	// ./beatmaps/{sid}/{sid}_nv_nb_ns.osz
	// ./beatmaps/{sid}/{sid}_nv_nh_ns.osz
	// ./beatmaps/{sid}/{sid}_nb_nh_ns.osz
	// ./beatmaps/{sid}/{sid}_nv_nb_nh_ns.osz
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		logger.Error(err)
		return
	}
	w := zip.NewWriter(&res)
	defer func() {
		err = w.Close()
		if err != nil {
			logger.Error(err)
		}
	}()
	skip := false
	for _, f := range r.File {
		skip = false
		f.FileInfo()
		for _, r2 := range notInRegexp {
			if r2.MatchString(f.Name) {
				skip = true
				logger.Debugf("skiped file:'%s'", f.Name)
				break
			}
		}
		if skip {
			continue
		}
		err = func() (err error) {
			rc, err := f.OpenRaw()
			if err != nil {
				logger.Fatalf("Failed to open file %s: %v", f.Name, err)
				return
			}

			fw, err := w.CreateRaw(&f.FileHeader)
			if err != nil {
				logger.Fatalf("Failed to create entry for %s: %v", f.Name, err)
				return
			}
			_, err = io.Copy(fw, rc)
			if err != nil {
				logger.Fatalf("Failed to write entry for %s: %v", f.Name, err)
			}
			return
		}()

	}
	return
}
