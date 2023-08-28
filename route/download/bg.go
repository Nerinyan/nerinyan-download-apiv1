package download

//func BearmapBg(c echo.Context) (err error) {
//
//    req, err := checkRequestIsSetOrBeatmap()
//    if err != nil {
//        return nil, err
//    }
//
//    if req[0] == "error" {
//        return newJSONErrorResponse("I can't to specify if what you requested is beatmapset id or beatmap id. But if you requested beatmapset id, add '-' before beatmapset id.", http.StatusNotFound)
//    }
//
//    r := regexp.MustCompile(`(\[[0-9]K\] )`)
//    if matches := r.FindStringSubmatch(req[2]); len(matches) > 0 {
//        req[2] = strings.Replace(req[2], matches[0], "", 1)
//    }
//
//    // beatmap file exist check
//    if err := checkFile(req[1]); err != nil {
//        return newJSONErrorResponse("An error occurred while trying to find BG.", http.StatusNotFound)
//    }
//
//    unzipDir := fmt.Sprintf("%s/%d", ROOT_UNZIP, req[1])
//    if _, err := os.Stat(unzipDir); os.IsNotExist(err) {
//        if err := unzipFile(req[1], false, ROOT_BEATMAP, ROOT_UNZIP); err != nil {
//            return newJSONErrorResponse("An error occurred while trying to find BG.", http.StatusNotFound)
//        }
//    }
//
//    fileROOT, err := getFileRoot(req[0] == "sid", req[1], req, ROOT_UNZIP)
//    if err != nil || fileROOT == "ERROR" {
//        return newJSONErrorResponse("An error occurred while trying to find BG.", http.StatusNotFound)
//    }
//
//    return newFileResponse(fileROOT), nil
//}
//
//func checkRequestIsSetOrBeatmap(bid string, beatmapid int, NERINYAN_API string, re bool) ([]interface{}, error) {
//    url := fmt.Sprintf("%s/search?q=%s&s=all&nsfw=true", NERINYAN_API, bid)
//    if beatmapid < 0 && !re {
//        url += "&option=s"
//    }
//    if re {
//        url += "&option=m"
//    }
//
//    resp, err := http.Get(url)
//    if err != nil {
//        return nil, err
//    }
//    defer resp.Body.Close()
//
//    if resp.StatusCode == 200 {
//        var body []map[string]interface{}
//        err := json.NewDecoder(resp.Body).Decode(&body)
//        if err != nil {
//            return nil, err
//        }
//
//        if len(body) > 1 {
//            return checkRequestIsSetOrBeatmap(bid, beatmapid, NERINYAN_API, true)
//        }
//
//        if bid == body[0]["id"].(string) {
//            return []interface{}{"sid", body[0]["id"]}, nil
//        }
//
//        for _, bmap := range body[0]["beatmaps"].([]interface{}) {
//            bmapMap := bmap.(map[string]interface{})
//            if bid == bmapMap["id"].(string) {
//                return []interface{}{"bid", body[0]["id"], bmapMap["version"]}, nil
//            }
//        }
//    }
//
//    return nil, fmt.Errorf("could not process request")
//}
//
//func checkFile(bbid int, NERINYAN_API string, ROOT_BEATMAP string, ROOT_UNZIP string) error {
//    filePath := fmt.Sprintf("%s/%d.osz", ROOT_BEATMAP, bbid)
//    if _, err := os.Stat(filePath); os.IsNotExist(err) {
//        resp, err := http.Get(fmt.Sprintf("%s/d/%d", NERINYAN_API, bbid))
//        if err != nil {
//            return err
//        }
//        defer resp.Body.Close()
//
//        out, err := os.Create(fmt.Sprintf("%s/%d.osz", ROOT_UNZIP, bbid))
//        if err != nil {
//            return err
//        }
//        defer out.Close()
//
//        _, err = io.Copy(out, resp.Body)
//        if err != nil {
//            return err
//        }
//
//        // Assuming unzipfile is another function in your program
//        // await unzipfile(bbid=bbid, istemp=True)
//        // Uncomment and replace this line with the equivalent synchronous Golang function for unzipping files
//    }
//    return nil
//}
//
//func unzipFile(bbid int, istemp bool, ROOT_BEATMAP string, ROOT_UNZIP string) error {
//    var zipFilePath string
//    if !istemp {
//        zipFilePath = fmt.Sprintf("%s/%d.osz", ROOT_BEATMAP, bbid)
//    } else {
//        zipFilePath = fmt.Sprintf("%s/%d.osz", ROOT_UNZIP, bbid)
//    }
//
//    r, err := zip.OpenReader(zipFilePath)
//    if err != nil {
//        return err
//    }
//    defer r.Close()
//
//    extractPath := fmt.Sprintf("%s/%d", ROOT_UNZIP, bbid)
//
//    for _, f := range r.File {
//        rc, err := f.Open()
//        if err != nil {
//            return err
//        }
//
//        fpath := filepath.Join(extractPath, f.Name)
//        if f.FileInfo().IsDir() {
//            os.MkdirAll(fpath, os.ModePerm)
//        } else {
//            var dir string
//            if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
//                dir = fpath[:lastIndex]
//                os.MkdirAll(dir, f.Mode())
//            }
//            f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
//            if err != nil {
//                return err
//            }
//
//            _, err = io.Copy(f, rc)
//            f.Close()
//            if err != nil {
//                return err
//            }
//        }
//        rc.Close()
//    }
//    return nil
//}
//
//func getFileRoot(isBeatmap bool, bbid int, req []string, ROOT_UNZIP string) (string, error) {
//    owd, err := os.Getwd()
//    if err != nil {
//        return "", err
//    }
//
//    unzipDir := fmt.Sprintf("%s/%d/", ROOT_UNZIP, bbid)
//    err = os.Chdir(unzipDir)
//    if err != nil {
//        return "", err
//    }
//
//    defer os.Chdir(owd) // return to original directory regardless of exit
//
//    files, err := ioutil.ReadDir("./")
//    if err != nil {
//        return "", err
//    }
//
//    for _, file := range files {
//        if !file.IsDir() {
//            fileName := file.Name()
//
//            if !isBeatmap {
//                if strings.HasSuffix(fileName, ".png") || strings.HasSuffix(fileName, ".jpg") || strings.HasSuffix(fileName, ".jpeg") {
//                    return filepath.Join(unzipDir, fileName), nil
//                }
//            } else {
//                if strings.Contains(fileName, req[2]) {
//                    content, err := ioutil.ReadFile(fileName)
//                    if err != nil {
//                        return "", err
//                    }
//
//                    r := regexp.MustCompile(`(?<=0,0,").+(?=",0,0)`)
//                    matches := r.FindStringSubmatch(string(content))
//                    if len(matches) > 0 {
//                        imgLine := matches[0]
//                        return filepath.Join(unzipDir, imgLine), nil
//                    }
//                }
//            }
//        }
//    }
//
//    return "ERROR", nil
//}
