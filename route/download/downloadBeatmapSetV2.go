package download

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"github.com/Nerinyan/nerinyan-download-apiv1/config"
	"github.com/Nerinyan/nerinyan-download-apiv1/db/mariadb"
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"github.com/Nerinyan/nerinyan-download-apiv1/osu"
	"github.com/Nerinyan/nerinyan-download-apiv1/utils"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func (oszReq) TableName() string {
	return _TB_BEATMAPSET
}

type oszReq struct {
	Sid             int       `param:"setId"`
	Mid             int       `param:"mapId"`
	NoVideo         bool      `query:"noVideo"`
	NoBackground    bool      `query:"noBg"`
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

func (v *oszReq) IsNoVideo() bool {
	return v.Nv || v.NoVideo
}
func (v *oszReq) IsNoBackground() bool {
	return v.Nb || v.NoBackground
}
func (v *oszReq) IsNoHitsound() bool {
	return v.Nh || v.NoHitsound
}
func (v *oszReq) IsNoStoryboard() bool {
	return v.Ns || v.NoStoryboard
}
func (v *oszReq) isModify() bool {
	return v.IsNoVideo() || v.IsNoBackground() || v.IsNoHitsound() || v.IsNoStoryboard()
}

func (v *oszReq) getSourceFileName() (path string) {
	path = fmt.Sprintf("%s/%d/%d", config.Config.TargetDir, v.BeatmapsetId, v.BeatmapsetId)
	path += "." + _SERVER_OSZ_EXT
	return
}

func (v *oszReq) getOptionFileName() (path string) {
	path = fmt.Sprintf("%s/%d/%d", config.Config.TargetDir, v.BeatmapsetId, v.BeatmapsetId)
	var args []string

	if v.IsNoVideo() {
		args = append(args, "nv")
	}
	if v.IsNoBackground() {
		args = append(args, "nb")
	}
	if v.IsNoHitsound() {
		args = append(args, "nh")
	}
	if v.IsNoStoryboard() {
		args = append(args, "ns")
	}
	if len(args) > 0 {
		path += "_" + strings.Join(args, "_")
	}
	path += "." + _SERVER_OSZ_EXT
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
		logger.Infof("nv: %t, nb: %t, nh: %t, ns: %t", req.IsNoVideo(), req.IsNoBackground(), req.IsNoHitsound(), req.IsNoStoryboard())
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

		//nv, nh, nb, ns
		rd, e := rebuildOsz(data, req.IsNoVideo(), req.IsNoHitsound(), req.IsNoBackground(), req.IsNoStoryboard())
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

func rebuildOsz(data []byte, nv, nh, nb, ns bool) (res bytes.Buffer, err error) {
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
	background := map[string]bool{}
	video := map[string]bool{}
	storyBoard := map[string]bool{}
	hitSound := map[string]bool{}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !_REGEXP_OSU_FILES.MatchString(f.FileHeader.Name) {
			continue
		}
		func() {
			rc, err := f.Open()
			if err != nil {
				logger.Fatalf("Failed to open file %s: %v", f.Name, err)
				return
			}
			defer rc.Close()
			files := osu.ParseOsuFileInfo(rc)
			for _, s := range files.Background {
				background[s] = true
			}
			for _, s := range files.Video {
				video[s] = true
			}
			for _, s := range files.StoryBoard {
				storyBoard[s] = true
			}
			for _, s := range files.HitSound {
				hitSound[s] = true
			}
		}()

	}

	for _, f := range r.File {
		//nv, nh, nb, ns
		if nv && video[f.FileHeader.Name] {
			//logger.Debugf("skip file: %s", f.FileHeader.Name)
			continue
		}
		if nh && (hitSound[f.FileHeader.Name] || _REGEXP_NH.MatchString(f.FileHeader.Name)) {
			//logger.Debugf("skip file: %s", f.FileHeader.Name)
			continue
		}
		if nb && background[f.FileHeader.Name] {
			//logger.Debugf("skip file: %s", f.FileHeader.Name)
			continue
		}
		if ns && (storyBoard[f.FileHeader.Name] || _REGEXP_NS.MatchString(f.FileHeader.Name)) {
			//logger.Debugf("skip file: %s", f.FileHeader.Name)
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
