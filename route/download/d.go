package download

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/Nerinyan/nerinyan-download-apiv1/config"
	"github.com/Nerinyan/nerinyan-download-apiv1/db/mariadb"
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	_REGEXP_NV, _ = regexp.Compile(`[.](mp4|m4v)$`)
	_REGEXP_NH, _ = regexp.Compile(`[.]osb$`)
	_REGEXP_NB, _ = regexp.Compile(`^(normal-|nightcore-|drum-|soft-|spinnerspin)`)
	_REGEXP_NS, _ = regexp.Compile("[.](png|jpg)$")
)

type _req struct {
	BeatmapSetId int  `param:"setId"`
	Novideo      bool `query:"noVideo"`
	Nobackground bool `query:"noBg"`
	NoHitsound   bool `query:"noHitsound"`
	NoStoryboard bool `query:"noStoryboard"`
	Nv           bool `query:"nv"`
	Nh           bool `query:"nh"`
	Nb           bool `query:"nb"`
	Nsb          bool `query:"nsb"`
}

func (v *_req) parse() (sid int, nv, nb, nh, ns bool) {
	sid = v.BeatmapSetId
	nv = v.Nv || v.Novideo
	nb = v.Nb || v.Nobackground
	nh = v.Nh || v.NoHitsound
	ns = v.Nsb || v.NoStoryboard
	return
}

type _BeatmapSet struct {
	BeatmapsetId int       `json:"id" gorm:"column:BEATMAPSET_ID"`
	Artist       string    `json:"artist" gorm:"column:ARTIST"`
	Creator      string    `json:"creator" gorm:"column:CREATOR"`
	Title        string    `json:"title" gorm:"column:TITLE"`
	LastUpdated  time.Time `json:"last_updated" gorm:"column:LAST_UPDATED"`
}

func (_BeatmapSet) TableName() string {
	return "BEATMAPSET"
}
func (v *_BeatmapSet) GetServerPath() string {
	return fmt.Sprintf("%s/%d/%d.zip", config.Config.TargetDir, v.BeatmapsetId, v.BeatmapsetId)
}
func (v *_BeatmapSet) GetClientFilename() string {
	return cannotUseFilename.ReplaceAllString(fmt.Sprintf("%d %s - %s.osz", v.BeatmapsetId, v.Artist, v.Title), "_")
}

func D(c echo.Context) (err error) {

	var req _req
	err = c.Bind(&req)
	if err != nil {
		logger.Error(err)
		return
	}
	var set _BeatmapSet
	err = mariadb.Mariadb.Model(&_BeatmapSet{}).Where(&_BeatmapSet{BeatmapsetId: req.BeatmapSetId}).Find(&set).Error
	if err != nil {
		logger.Error(err)
		return
	}
	var buf chan []byte
	var length int64

	if true { // 나중에 반쵸 api 콜수 체크하는 로직 추가
		buf, length, err = downloadFromBancho(set.BeatmapsetId)
		if err != nil {
			logger.Error(err)

		}
	}

	if err != nil || false { // 반쵸에서 에러가 발생했거나 || 반쵸를 스킵하는경우
		buf, length, err = downloadFromBeatconnect(set.BeatmapsetId)
		if err != nil {
			logger.Error(err)
			return
		}
	}

	// ======================================
	// 클라이언트 응답.
	logger.Info(echo.HeaderContentLength, strconv.FormatInt(length, 10))
	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(length, 10))
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, set.GetClientFilename()))
	c.Response().Header().Set(echo.HeaderContentType, "application/x-osu-beatmap-archive")
	clientError := false
	var b bytes.Buffer
	for ch := range buf {
		b.Write(ch) // 버퍼에 쓸때는 에러가 안난다고 봐도 무방함 (OOM 은 발생 가능함)
		if !clientError {
			_, err := c.Response().Write(ch)
			if err != nil {
				logger.Error(err)
				clientError = true
			}
		}
	}
	if int64(b.Len()) != length {
		return fmt.Errorf("contentLength: %d, RX bytes: %d download failed", length, int64(b.Len()))
	}
	c.Response().Flush()
	data := b.Bytes()
	// ======================================
	// 로컬 파일로 저장
	err = saveLocal2(&data, set.GetServerPath())
	if err != nil {
		logger.Error(err)
	}

	sid, nv, nb, nh, ns := req.parse()
	if nv || nb || nh || ns {
		logger.Infof("nv: %t, nb: %t, nh: %t, ns: %t", nv, nb, nh, ns)
		path, reg := buildFilename(sid, nv, nb, nh, ns)
		rd, err := rebuildOsz(&data, reg)
		if err != nil {
			logger.Error(err)
		}
		err = saveLocal2(&rd, fmt.Sprintf("%s/%s", config.Config.TargetDir, path))
		if err != nil {
			logger.Error(err)
		}
	}

	return

}

func rebuildOsz(data *[]byte, notInRegexp []*regexp.Regexp) (res []byte, err error) {
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
	r, err := zip.NewReader(bytes.NewReader(*data), int64(len(*data)))
	if err != nil {
		logger.Error(err)
		return
	}
	var bufDest bytes.Buffer
	w := zip.NewWriter(&bufDest)
	defer func() {
		_ = w.Flush()
		_ = w.Close()
		res = bufDest.Bytes()
	}()
	skip := false
	for _, f := range r.File {
		skip = false
		f.FileInfo()
		for _, r2 := range notInRegexp {
			if r2.MatchString(f.Name) {
				skip = true
				logger.Infof("skiped file:'%s'", f.Name)
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
			fw, err := w.Create(f.Name)
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

// 내부적으로 비동기 처리됨
func downloadFromBancho(beatmapSetId int) (buf chan []byte, length int64, initError error) {

	client := &http.Client{Timeout: time.Second * 10}
	url := fmt.Sprintf("https://osu.ppy.sh/api/v2/beatmapsets/%d/download", beatmapSetId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Add("Authorization", getToken())
	res, err := client.Do(req)
	if err != nil {
		return
	}
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("%s request error %s", req.URL, res.Status)
		return
	}
	//===========================================
	length = res.ContentLength
	buf = make(chan []byte)
	go func() {
		defer close(buf)
		defer res.Body.Close()

		//==========================
		for {
			tmp := make([]byte, 16384)
			n, e := res.Body.Read(tmp)
			if n > 0 {
				buf <- tmp[:n]
			}
			if e == io.EOF {
				return
			} else if e != nil { //에러처리
				logger.Error(e)
				return
			}
		}

	}()
	return
}

// 내부적으로 비동기 처리됨
func downloadFromBeatconnect(beatmapSetId int) (buf chan []byte, length int64, initError error) {

	client := &http.Client{Timeout: time.Second * 10}
	url := fmt.Sprintf("https://beatconnect.io/b/%d", beatmapSetId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("%s request error %s", req.URL, res.Status)
		return
	}
	//===========================================
	length = res.ContentLength
	buf = make(chan []byte)
	go func() {
		defer close(buf)
		defer res.Body.Close()

		//==========================
		tmp := make([]byte, 64000)
		for {
			n, e := res.Body.Read(tmp)
			if n > 0 {
				buf <- tmp[:n]
			}
			if e == io.EOF {
				return
			} else if e != nil { //에러처리
				logger.Error(e)
				return
			}
		}

	}()
	return
}

func getToken() (token string) {
	return config.Config.Osu.Token.TokenType + " " + config.Config.Osu.Token.AccessToken
}

func buildFilename(sid int, nv, nb, nh, ns bool) (path string, regexps []*regexp.Regexp) {
	path = fmt.Sprintf("%d/%d", sid, sid)
	var args []string

	if nv {
		args = append(args, "nv")
		regexps = append(regexps, _REGEXP_NV)
	}
	if nb {
		args = append(args, "nb")
		regexps = append(regexps, _REGEXP_NB)
	}
	if nh {
		args = append(args, "nh")
		regexps = append(regexps, _REGEXP_NH)
	}
	if ns {
		args = append(args, "ns")
		regexps = append(regexps, _REGEXP_NS)
	}
	if len(args) > 0 {
		path += "_" + strings.Join(args, "_")
	}
	path += ".zip"
	return

}

func clean(setId int) error {
	return os.RemoveAll(fmt.Sprintf("%s/%d/*", config.Config.TargetDir, setId))
}

func create(path string) (*os.File, error) {
	dir := filepath.Dir(path)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}

	return os.Create(path)
}

func saveLocal2(data *[]byte, path string) (err error) {
	tmp := path + strconv.FormatInt(time.Now().UnixNano(), 16)
	func() {
		f, err := create(tmp)
		if err != nil {
			return
		}
		defer f.Close()
		_, err = f.Write(*data)
		if err != nil {
			return
		}
	}()

	return os.Rename(tmp, path)
}
