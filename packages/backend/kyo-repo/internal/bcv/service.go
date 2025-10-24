package bcv

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cespare/xxhash"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/util"
)

const MetadataProcessedKey = "processed"
const MetadataLastProcessedKey = "lastprocessed"
const MetadataRatesParsedKey = "ratesparsed"
const MetadataNumOfSheetsKey = "numofsheets"

type Service struct {
	ctx        context.Context
	bucketName string
	url        string
	filePath   string
	s3Client   *s3.Client
	httpClient *http.Client
}

func NewService(ctx context.Context) (*Service, error) {
	bucketName, err := GetBcvBucket()
	if err != nil {
		return nil, err
	}

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
		bucketName: bucketName,
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

	links, err := service.allFileLinks()

	if err != nil {
		return err
	}

	length := len(links)
	if length == 0 {
		log.Println("No files to check")
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(length)
	errorChan := make(chan error, length)

	for pos, link := range links {
		go func() {
			defer wg.Done()
			linkErr := service.checkLink(pos, link)
			if linkErr != nil {
				errorChan <- fmt.Errorf("error checking file link %d - %s: %s", pos, link, linkErr)
			}
		}()
	}

	wg.Wait()
	close(errorChan)

	return util.HasErrors(errorChan)
}

func (service Service) checkLink(pos int, link string) error {
	fileName := link[strings.LastIndex(link, "/")+1:]
	objectKey := fmt.Sprintf("rates/bcv=%d=%s", pos, fileName)

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return err
	}

	headObj, err := service.s3Client.HeadObject(service.ctx, &s3.HeadObjectInput{
		Bucket: aws.String(service.bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		//err.Error().contains("The specified key does not exist")
		is404 := strings.Contains(err.Error(), "response error StatusCode: 404")

		if !is404 {
			return err
		}

	} else {

		oldEtag := headObj.Metadata["etag"]
		oldLastModified := headObj.Metadata["lastmodified"]
		if oldEtag != "" && oldLastModified != "" {
			req.Header.Add("If-None-Match", oldEtag)
			req.Header.Add("If-Modified-Since", oldLastModified)
		}
	}

	res, err := service.httpClient.Do(req)
	//log.Errorf("Downloaded: %s %v", processor.Filepath, wgErr)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing response body:", err)
			return
		}
	}(res.Body)

	if res.StatusCode == 304 {
		log.Printf("File %s is up to date", objectKey)
		return nil
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("error downloading file %s: status code %d", link, res.StatusCode)
	}

	//hash, err := FileHash(res.Body)
	//if err != nil {
	//	return err
	//}

	etag := res.Header.Get("ETag")

	metadata := make(map[string]string)
	metadata["etag"] = etag
	metadata["lastmodified"] = res.Header.Get("Last-Modified")
	metadata["url"] = link

	if headObj == nil || headObj.Metadata[MetadataProcessedKey] == "" {
		metadata[MetadataProcessedKey] = "false"
	}

	_, err = service.s3Client.PutObject(service.ctx, &s3.PutObjectInput{
		Bucket:            aws.String(service.bucketName),
		Key:               aws.String(objectKey),
		Body:              res.Body,
		ChecksumAlgorithm: types.ChecksumAlgorithmCrc64nvme,
		//ChecksumCRC32:             nil,
		//ChecksumCRC32C:            nil,
		//ChecksumSHA1:              nil,
		//ChecksumSHA256:            nil,
		ContentLength: &res.ContentLength,
		//ContentType:                 res.ty,
		Metadata: metadata,
	})

	if err != nil {
		return err
	}

	return nil
}

func (service Service) allFileLinks() ([]string, error) {

	historicFilesUrl := service.url + service.filePath

	var links []string
	var pagelinks []string
	err := service.historicLinks(&links, historicFilesUrl)
	pagelinks = append(pagelinks, historicFilesUrl)

	if err != nil {
		return nil, err
	}

	checkLinks := func() (bool, error) {
		execute := false
		for _, link := range links {
			if !strings.HasSuffix(link, ".xls") {
				nextUrl := service.url + link
				if !slices.Contains(pagelinks, nextUrl) {
					err := service.historicLinks(&links, nextUrl)
					execute = true
					pagelinks = append(pagelinks, nextUrl)
					if err != nil {
						return execute, err
					}
				}
			}
		}

		return execute, nil
	}

	while := true
	counter := 0
	for while {
		counter++
		while, err = checkLinks()
		if err != nil {
			return nil, err
		}
	}

	links = slices.DeleteFunc(links, func(link string) bool {
		return !strings.HasSuffix(link, ".xls")
	})
	slices.Reverse(links)

	return links, nil
}

func (service Service) historicLinks(links *[]string, pageUrl string) error {

	res, err := service.httpClient.Get(pageUrl)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing response body:", err)
			return
		}
	}(res.Body)

	if res.StatusCode != 200 {
		return fmt.Errorf("error bcv page res: %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}

	section := doc.Find("#block-system-main")

	if section == nil {
		log.Printf("#block-system-main Section not found")
		return fmt.Errorf("error getting section")
	}

	sel := section.Find("a")
	for i := range sel.Nodes {
		single := sel.Eq(i)
		href, b := single.Attr("href")
		if b {
			if !slices.Contains(*links, href) {
				*links = append(*links, href)
			}
		}
	}

	return nil
}

func FileHash(body io.ReadCloser) (int64, error) {

	buf := make([]byte, 1024*1024)
	hash := xxhash.New()
	if _, err := io.CopyBuffer(hash, body, buf); err != nil {
		return 0, err
	}
	bytesSum := hash.Sum(nil)
	fileHash := int64(xxhash.Sum64(bytesSum))
	return fileHash, nil
}
