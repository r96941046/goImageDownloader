package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/r96941046/imageDownloader/config"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func parseArgs() (string, error) {

	// dir is a string pointer
	dir := flag.String("dir", "Files", "dir for fetching filenames and download target")

	flag.Parse()

	// use dir value here
	if *dir == "" {
		return "", errors.New("argument `-dir` is required")
	}

	// get current directory
	// os.Args[0] is bin/speechDowloader
	// filepath.Dir(os.Args[0]) is bin/
	cwdir, err := filepath.Abs(filepath.Dir(os.Args[0]))

	if err != nil {
		fmt.Println("Cannot get current working directory:", err)
		os.Exit(1)
	}

	downloadDir := filepath.Join(cwdir, *dir)

	if _, err := os.Stat(downloadDir); err != nil {
		if os.IsNotExist(err) {
			// dir does not exists, mkdir
			os.Mkdir(downloadDir, 0777)
		} else {
			// other error
			fmt.Println("Check download dir failed:", err)
			os.Exit(1)
		}

	}

	return downloadDir, nil
}

func getLinks() []string {

	client := http.Client{}

	req, err := http.NewRequest("GET", config.AlbumLink, nil)
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var jsonmap map[string]*json.RawMessage
	var images []map[string]interface{}

	err = json.Unmarshal(body, &jsonmap)
	err = json.Unmarshal(*jsonmap["data"], &images)

	var links []string
	for _, image := range images {
		links = append(links, image["link"].(string))
	}

	return links
}

func downloadLinks(links []string, dir string) {

	fmt.Println("Start downloading images...")

	channel := make(chan int64)

	for _, link := range links {

		imageUrl, err := url.Parse(link)
		if err != nil {
			fmt.Println("Parse url failed:", err)
		}

		segments := strings.Split(imageUrl.Path, "/")
		fileName := segments[len(segments)-1]
		downloadPath := filepath.Join(dir, fileName)

		go download(link, downloadPath, channel)
	}

	var totalBytes int64
	for i := 0; i < len(links); i++ {
		totalBytes += <-channel
	}

	fmt.Printf("Done, %v bytes of images downloaded", totalBytes)
}

func download(link string, downloadPath string, channel chan int64) {

	file, err := os.Create(downloadPath)
	if err != nil {
		fmt.Println("Create file failed:", err)
	}

	defer file.Close()

	client := http.Client{}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		fmt.Println("Download image failed:", err)
	}

	resp, err := client.Do(req)

	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		fmt.Println("Copy image to file failed:", err)
	}

	channel <- size
}

func main() {

	dir, err := parseArgs()

	if err != nil {
		fmt.Println("Parse argument failed:", err)
		os.Exit(1)
	}

	downloadLinks(getLinks(), dir)
}
