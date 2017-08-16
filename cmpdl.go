package main

import (
	"io/ioutil"
	"encoding/json"
	"time"
	"os"
	"fmt"
	"path/filepath"
	"strconv"
	"net/http"
	"io"
	"crypto/tls"
	"path"
	"strings"
)

type ModFile struct {
	FileID    int  `json:"fileID"`
	ProjectID int  `json:"projectID"`
	Required  bool `json:"required"`
}

type Manifest struct {
	Author          string `json:"author"`
	Files           []ModFile `json:"files"`
	ManifestType    string `json:"manifestType"`
	ManifestVersion int    `json:"manifestVersion"`
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
	Url string
	Err bool
}

var client *http.Client
var modpackPath string

func main() {
	dat, err := ioutil.ReadFile("./manifest.json")
	check(err)
	var jsonData Manifest
	json.Unmarshal(dat, &jsonData)
	modepackDirName := jsonData.Name + "-" + strconv.FormatInt(time.Now().Unix()/1000, 10)
	modpackPath = filepath.Join(".", modepackDirName)
	os.MkdirAll(modpackPath, os.ModePerm)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}
	jobs := make(chan ModFile, 100)
	results := make(chan Result, 100)
	for w := 1; w <= 3; w++ {
		go worker(jobs, results)
	}
	for _, mod := range jsonData.Files {
		jobs <- mod
	}
	close(jobs)
	for a := 1; a <= len(jsonData.Files); a++ {
		s := <-results
		if (s.Err) {
			fmt.Printf("%s\n", s.Url)
		}
	}
}

func worker(jobs <-chan ModFile, results chan<- Result) {

	for file := range jobs {

		baseUrl := fmt.Sprintf("http://minecraft.curseforge.com/projects/%v/files/%v/download", file.ProjectID, file.FileID)
		var finalUrl2 string
		for i := 0; i < 5; i++ {
			finalUrl, err := getLocationHeader(baseUrl);
			if err == nil {
				finalUrl2 = finalUrl
				break
			} else {
				if i == 4 {
					res := &Result{baseUrl, true}
					results <- *res
					return
				}
			}
		}

		res := &Result{finalUrl2, false}
		results <- *res
	}
}

func getLocationHeader(baseUrl string) (string, error) {
	userAgent := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/53.0.2785.143 Chrome/53.0.2785.143 Safari/537.36";
	fmt.Println("downloading:" + baseUrl)
	req, _ := http.NewRequest("GET", baseUrl, nil)
	req.Header.Set("User-Agent", userAgent);
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	defer res.Body.Close()
	var finalUrl = res.Request.URL.String()
	sp := strings.Split(finalUrl, "/")
	var fileName = sp[len(sp)-1]
	fmt.Println("file name:" + fileName)
	if len(fileName) == 0 {
		fileName = time.Now().String()
	}
	path := path.Join(modpackPath, fileName)
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("create file err : " + path)
		return res.Request.URL.String(), err;
	}
	defer f.Close()
	io.Copy(f, res.Body);
	fmt.Println("copy finish")
	return res.Request.URL.String(), nil;
}
