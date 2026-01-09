package bcv

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
)

const MetadataProcessedKey = "processed"
const MetadataLastProcessedKey = "lastprocessed"
const MetadataRatesParsedKey = "ratesparsed"
const MetadataNumOfSheetsKey = "numofsheets"

type Service struct {
	ctx        context.Context
	url        string
	filePath   string
	s3Client   *s3.Client
	httpClient *http.Client
}

func NewService(ctx context.Context) (*Service, error) {

	bcvUrlSecret, err := resource.Get("SecretBcvUrl", "value")
	if err != nil {
		log.Printf("Error getting bcv url: %s", err)
		return nil, fmt.Errorf("error getting bcv url: %s", err)
	}
	filePath, err := resource.Get("SecretBcvFileStartPath", "value")
	if err != nil {
		log.Printf("Error getting bcv file start path: %s", err)
		return nil, fmt.Errorf("error getting bcv file start path: %s", err)
	}

	client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	netTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: time.Second * 10,
		}).DialContext,
		TLSHandshakeTimeout: time.Second * 8,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}

	netClient := &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}

	return &Service{
		ctx:        ctx,
		url:        bcvUrlSecret.(string),
		filePath:   filePath.(string),
		s3Client:   client,
		httpClient: netClient,
	}, nil
}

type FileInfo struct {
	Pos       int    `json:"pos"`
	Url       string `json:"url"`
	Size      int64  `json:"size"`
	Etag      string `json:"etag"`
	Hash      int64  `json:"hash"`
	EtagWorks bool   `json:"etagWorks"`
}

func (service Service) Check() error {

	return nil

	//links, err := service.FileLinks()
	//
	//if err != nil {
	//	return err
	//}
	//
	//length := len(links)
	//if length == 0 {
	//	log.Println("No files to check")
	//	return nil
	//}
	//
	//var wg sync.WaitGroup
	//wg.Add(length)
	//errorChan := make(chan error, length)
	//
	//for pos, link := range links {
	//	go func() {
	//		defer wg.Done()
	//		//linkErr := service.checkLink(pos, link)
	//		//linkErr := service.processLink(link, true)
	//		//if linkErr != nil {
	//		//	errorChan <- fmt.Errorf("error checking file link %d - %s: %s", pos, link, linkErr)
	//		//}
	//	}()
	//}
	//
	//wg.Wait()
	//close(errorChan)
	//
	//return util.HasErrors(errorChan)
}

type DownloadResult struct {
	Link         string
	Etag         string
	LastModified string
	FilePath     string
	FileSize     int64
	Error        error
}

func (service Service) Download(bcvFile *model.BcvFiles, link string) DownloadResult {
	fileName := link[strings.LastIndex(link, "/")+1:]
	filePath := util.TmpFileName(util.UuidV7() + fileName)

	result := DownloadResult{}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		result.Error = err
		return result
	}

	if bcvFile != nil {
		req.Header.Add("If-None-Match", bcvFile.Etag)
		req.Header.Add("If-Modified-Since", bcvFile.LastModified)
	}

	res, err := service.httpClient.Do(req)
	//log.Errorf("Downloaded: %s %v", processor.Filepath, wgErr)
	if err != nil {
		result.Error = err
		return result
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("error closing response body:", err)
			return
		}
	}(res.Body)

	if res.StatusCode == 304 {
		log.Printf("File %s is up to date", link)
		return result
	}

	if res.StatusCode != 200 {
		result.Error = fmt.Errorf("error downloading file %s - %s", link, res.Status)
		return result
	}

	file, err := os.Create(filePath)
	if err != nil {
		result.Error = fmt.Errorf("error creating file %s - %s", filePath, err)
		return result
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	_, err = io.Copy(file, res.Body)
	if err != nil {
		result.Error = fmt.Errorf("error saving file %s: %s", filePath, err)
		return result
	}

	result.Link = link
	result.Etag = res.Header.Get("ETag")
	result.LastModified = res.Header.Get("Last-Modified")
	result.FilePath = filePath
	result.FileSize = res.ContentLength

	return result
}

func (service Service) FileLinks(last bool) ([]string, error) {
	var visited []string
	var fileLinks []string
	var nextPages []string

	nextPages = append(nextPages, service.url+service.filePath)

	running := true
	for running {
		length := len(nextPages)
		if length == 0 {
			running = false
			break
		}

		var wg sync.WaitGroup
		wg.Add(length)
		linkChan := make(chan []string, length)
		errorChan := make(chan error, length)

		for _, pageUrl := range nextPages {
			go func() {
				defer wg.Done()
				links, err := service.links(pageUrl)
				if err != nil {
					errorChan <- err
					return
				}

				//log.Printf("Found %d links on page %s", len(links), pageUrl)
				linkChan <- links
				//log.Printf("Processed page: %s", pageUrl)
			}()

			visited = append(visited, pageUrl)
		}

		wg.Wait()
		close(linkChan)
		close(errorChan)

		err := util.HasErrors(errorChan)
		if err != nil {
			return nil, err
		}

		nextPages = nil

	ChanLoop:
		for links := range linkChan {
			for _, link := range links {
				pageLink := service.url + link
				if strings.HasSuffix(link, ".xls") {
					if !slices.Contains(fileLinks, link) {
						fileLinks = append(fileLinks, link)
						if last {
							running = false
							break ChanLoop
						}
					}
				} else {
					if !slices.Contains(visited, pageLink) && !slices.Contains(nextPages, pageLink) {
						nextPages = append(nextPages, pageLink)
					}
				}

			}
		}
	}

	return fileLinks, nil
}

func (service Service) links(pageUrl string) ([]string, error) {

	log.Printf("Fetching links from %s", pageUrl)

	res, err := service.httpClient.Get(pageUrl)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing response body:", err)
			return
		}
	}(res.Body)

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error bcv page res: %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	section := doc.Find("#block-system-main")

	if section == nil {
		log.Printf("#block-system-main Section not found")
		return nil, fmt.Errorf("error getting section")
	}

	sel := section.Find("a")
	var links []string
	for i := range sel.Nodes {
		single := sel.Eq(i)
		href, b := single.Attr("href")
		if b {
			if !slices.Contains(links, href) {
				links = append(links, href)
			}
		}
	}

	return links, nil
}
