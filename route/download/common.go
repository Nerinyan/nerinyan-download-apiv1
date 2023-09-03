package download

import (
	"fmt"
	"github.com/Nerinyan/nerinyan-download-apiv1/config"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

var downloadCount int
var mutex = sync.Mutex{}

func isLimitedDownload() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return downloadCount > 80
}
func callBancho() {
	mutex.Lock()
	defer mutex.Unlock()
	downloadCount++
}
func init() {
	ticker := time.NewTicker(time.Minute * 10)
	go func() {
		for ; ; <-ticker.C {
			mutex.Lock()
			downloadCount = 0
			mutex.Unlock()
		}
	}()
}

const _TB_BEATMAPSET = "BEATMAPSET"
const _TB_BEATMAP = "BEATMAPS"
const _SERVER_OSZ_EXT = "zip"

var _REGEXP_FN_NOT_ALLOW, _ = regexp.Compile(`([\\/:*?"<>|])`)
var _REGEXP_MANIA_KEY, _ = regexp.Compile(`\[[0-9]K]`)     // 시작부분만 역슬래시 붙혀줘도 됨
var _REGEXP_OSU_FILES, _ = regexp.Compile(`[.](osb|osu)$`) // 파싱 대상 파일확장자

var _REGEXP_NV, _ = regexp.Compile(`[.](mp4|m4v)$`)
var _REGEXP_NS, _ = regexp.Compile(`[.]osb$`)
var _REGEXP_NH, _ = regexp.Compile(`^(normal-|nightcore-|drum-|soft-|spinnerspin)`)
var _REGEXP_NB, _ = regexp.Compile("[.](png|jpg|jpeg)$")

func getSourceBGPath(sid, mid int) (path string) {
	path = fmt.Sprintf("%s/%d/", config.Config.TargetDir, sid)
	return
}

func getSourceOszPath(sid int) (path string) {
	path = fmt.Sprintf("%s/%d/%d", config.Config.TargetDir, sid, sid)
	path += "." + _SERVER_OSZ_EXT
	return
}

func downloadFromBancho2(beatmapSetId int) (reader io.ReadCloser, length int64, err error) {
	client := &http.Client{Timeout: time.Second * 60}
	url := fmt.Sprintf("https://osu.ppy.sh/api/v2/beatmapsets/%d/download", beatmapSetId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Add("Authorization", config.Config.Osu.Token.TokenType+" "+config.Config.Osu.Token.AccessToken)
	callBancho() // 반쵸 api 카운트
	res, err := client.Do(req)
	if err != nil {
		return
	}
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("%s request error %s", req.URL, res.Status)
		_ = res.Body.Close()
		return
	}
	//===========================================
	io.MultiReader()
	return res.Body, res.ContentLength, nil
}

func downloadFromBeatconnect2(beatmapSetId int) (reader io.ReadCloser, length int64, err error) {
	client := &http.Client{Timeout: time.Second * 60}
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
		_ = res.Body.Close()
		return
	}
	//===========================================
	return res.Body, res.ContentLength, nil
}
