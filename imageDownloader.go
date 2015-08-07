package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/r96941046/goImageDownloader/config"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
		log.Fatal(err)
	}

	downloadDir := filepath.Join(cwdir, *dir)

	if _, err := os.Stat(downloadDir); err != nil {
		if os.IsNotExist(err) {
			// dir does not exists, mkdir
			os.Mkdir(downloadDir, 0777)
		} else {
			// other error
			log.Fatal(err)
		}

	}

	return downloadDir, nil
}

func getLinks() []string {

	client := http.Client{}

	req, err := http.NewRequest("GET", config.AlbumLink, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var jsonmap map[string]*json.RawMessage
	var images []map[string]interface{}

	err = json.Unmarshal(body, &jsonmap)
	err = json.Unmarshal(*jsonmap["data"], &images)
	if err != nil {
		log.Fatal(err)
	}

	var links []string
	for _, image := range images {
		links = append(links, image["link"].(string))
	}

	return links
}

func downloadLinks(links []string, dir string) {

	fmt.Println("Start downloading images...")

	time.Sleep(time.Second * 1)

	start := time.Now()

	var wg sync.WaitGroup
	inputChannel := make(chan []string, len(links))
	outputChannel := make(chan int64)

	// pool of 8 goroutine workers
	// expecting tasks from inputChannel
	for i := 0; i < config.PoolSize; i++ {
		wg.Add(1)
		go worker(inputChannel, outputChannel, wg)
	}

	// start queuing tasks to gorouting pool
	for _, link := range links {

		imageUrl, err := url.Parse(link)
		if err != nil {
			log.Fatal(err)
		}

		segments := strings.Split(imageUrl.Path, "/")
		fileName := segments[len(segments)-1]
		downloadPath := filepath.Join(dir, fileName)

		input := []string{link, downloadPath}
		fmt.Println("Sending parameters to goroutine:", input)
		inputChannel <- input
	}

	// let the workers know when to stop
	close(inputChannel)

	// get task results from outputChannel
	var totalBytes int64
	for i := 0; i < len(links); i++ {
		totalBytes += <-outputChannel
	}

	// wait for the workers to finish
	// wg.Wait()

	elapsed := time.Since(start)

	fmt.Printf("Done, %v bytes of images downloaded\n", totalBytes)
	fmt.Printf("All tasks take %v to complete\n", elapsed)
	fmt.Printf("Each task takes %v to complete", elapsed/time.Duration(len(links)))
}

func worker(inputChannel chan []string, outputChannel chan int64, wg sync.WaitGroup) {

	defer wg.Done()

	for input := range inputChannel {
		download(input[0], input[1], outputChannel)
	}
}

func download(link string, downloadPath string, outputChannel chan int64) {

	fmt.Println("Downloading file:", link)

	file, err := os.Create(downloadPath)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	client := http.Client{}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	outputChannel <- size
}

func main() {

	dir, err := parseArgs()
	if err != nil {
		log.Fatal(err)
	}

	links := getLinks()
	if len(links) > 0 {
		downloadLinks(links, dir)
	} else {
		fmt.Println("No links were found")
	}
}
