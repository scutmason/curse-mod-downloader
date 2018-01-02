package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"errors"
)

type ModFile struct {
	FileID    int  `json:"fileID"`
	ProjectID int  `json:"projectID"`
	Required  bool `json:"required"`
}

type ModIndex struct {
	ModFile
	index int
}

type Manifest struct {
	Author          string    `json:"author"`
	Files           []ModFile `json:"files"`
	ManifestType    string    `json:"manifestType"`
	ManifestVersion int       `json:"manifestVersion"`
	Minecraft struct {
		ModLoaders []struct {
			ID      string `json:"id"`
			Primary bool   `json:"primary"`
		} `json:"modLoaders"`
		Version string `json:"version"`
	} `json:"minecraft"`
	Name      string `json:"name"`
	Overrides string `json:"overrides"`
	ProjectID int    `json:"projectID"`
	Version   string `json:"version"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}

}

type Result struct {
	URL string
	Err bool
}

var client *http.Client
var modpackPath string
var modpackDirName string
var total int

func main() {
	dat, err := ioutil.ReadFile("./manifest.json")
	check(err)
	var jsonData Manifest
	json.Unmarshal(dat, &jsonData)
	modepackDirName := jsonData.Name + "-" + strconv.FormatInt(time.Now().Unix()/1000, 10)
	modpackDirName = modepackDirName
	modpackPath = filepath.Join(".", modepackDirName)
	os.MkdirAll(modpackPath, os.ModePerm)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}
	jobs := make(chan ModIndex, 100)
	results := make(chan Result, 100)
	for w := 1; w <= 3; w++ {
		go worker(jobs, results)
	}
	total = len(jsonData.Files)
	for i, mod := range jsonData.Files {
		jobs <- ModIndex{mod,i+1}
	}
	close(jobs)
	for a := 1; a <= len(jsonData.Files); a++ {
		s := <-results
		if s.Err {
			writeError(s.URL, modepackDirName)
		}
	}
	if errorFile != nil {
		errorFile.Close()
	}
}

var errorFile *os.File

func writeError(s string, modepackDirName string) {
	if errorFile == nil {
		errorFile, _ = os.Create(modepackDirName + ".log")
	}
	errorFile.WriteString(s)
	errorFile.WriteString("\r\n")
}
func worker(jobs <-chan ModIndex, results chan<- Result) {

	for file := range jobs {
		baseURL := fmt.Sprintf("http://minecraft.curseforge.com/projects/%v/files/%v/download", file.ProjectID, file.FileID)
		var finalURL2 string
		for i := 0; i < 5; i++ {
			finalURL, err := getLocationHeader(baseURL, file.ProjectID, file.FileID, file.index)
			if err == nil {
				finalURL2 = finalURL
				break
			} else {
				if i == 4 {
					res := &Result{baseURL, true}
					results <- *res
					return
				}
			}
		}

		res := &Result{finalURL2, false}
		results <- *res
	}
}

func getLocationHeader(baseUrl string, projectId int, fileId int, index int) (string, error) {
	userAgent := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/53.0.2785.143 Chrome/53.0.2785.143 Safari/537.36"
	fmt.Println(strconv.Itoa(index) + "/" + strconv.Itoa(total) + "downloading:" + baseUrl)
	req, _ := http.NewRequest("GET", baseUrl, nil)
	req.Header.Set("User-Agent", userAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	if res.StatusCode > 300 || res.StatusCode < 200 {
		return "", errors.New("error network with code:" + strconv.Itoa(res.StatusCode))
	}
	defer res.Body.Close()
	var finalUrl = res.Request.URL.String()
	sp := strings.Split(finalUrl, "/")
	fileName, _ := url.QueryUnescape(sp[len(sp)-1])
	logToFile("file name:" + fileName + ",file id: " + strconv.Itoa(fileId) + ",project id: " + strconv.Itoa(projectId))
	if len(fileName) == 0 {
		fileName = time.Now().String()
	}
	modPath := path.Join(modpackPath, fileName)
	f, err := os.Create(modPath)
	if err != nil {
		fmt.Println("create file err : " + modPath)
		return res.Request.URL.String(), err
	}
	defer f.Close()
	io.Copy(f, res.Body)
	return res.Request.URL.String(), nil
}

func logToFile(s string) {
	//fmt.Println(s);
	//writeError(s, modpackDirName)
}
