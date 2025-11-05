package bcv_files

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yaz/kyo-repo/internal/bcv"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
)

func processAllFiles(ctx context.Context) error {
	service, err := bcv.NewService(ctx)

	if err != nil {
		return err
	}

	links, err := service.FileLinks()

	if err != nil {
		return err
	}

	length := len(links)
	if length == 0 {
		log.Println("No files to check")
		return nil
	}

	//var wg sync.WaitGroup
	//wg.Add(length)
	//resultChan := make(chan bcv.DownloadResult, length)
	//errorChan := make(chan error, length)
	//
	//for pos, link := range links {
	//
	//	go func(link string, pos int) {
	//		defer wg.Done()
	//
	//		result := service.Download(nil, link)
	//		if result.Error != nil {
	//			errorChan <- fmt.Errorf("error downloading file link %d - %s: %s", pos, link, result.Error)
	//			return
	//		}
	//
	//		resultChan <- result
	//
	//	}(link, pos)
	//}
	//
	//wg.Wait()
	//close(errorChan)
	//close(resultChan)
	//
	//err = util.HasErrors(errorChan)
	//if err != nil {
	//	return err
	//}
	//
	//for downloadResult := range resultChan {
	//	err = processResult(ctx, downloadResult)
	//	if err != nil {
	//		log.Printf("Error processing file %s: %s", downloadResult.Link, err)
	//		return err
	//	}
	//
	//}
	//
	//return nil

	var wg sync.WaitGroup
	wg.Add(length)
	errorChan := make(chan error, length)

	for pos, link := range links {

		go func(link string, pos int) {
			defer wg.Done()

			result := service.Download(nil, link)
			if result.Error != nil {
				errorChan <- fmt.Errorf("error downloading file link %d - %s: %s", pos, link, result.Error)
				return
			}

			err = processResult(ctx, result)
			if err != nil {
				errorChan <- fmt.Errorf("error processing file %s: %s", link, err)
				return
			}
		}(link, pos)
	}

	wg.Wait()
	close(errorChan)

	return util.HasErrors(errorChan)
}

func processResult(ctx context.Context, downloadResult bcv.DownloadResult) error {
	info := bcv.ParsingInfo{
		BucketKey:  downloadResult.Link,
		FilePath:   downloadResult.FilePath,
		ProcessAll: true,
		Ctx:        ctx,
	}

	result, err := info.Parse()
	if err != nil {
		return err
	}

	log.Printf("Processed %s rates parsed: %d inserted: %d", downloadResult.Link, result.Parsed, result.Inserted)

	repo := NewRepository(ctx)

	_, err = repo.Insert(model.BcvFiles{
		Link:         &downloadResult.Link,
		RateCount:    int64(result.Parsed),
		SheetCount:   int32(result.NumOfSheets),
		FileSize:     downloadResult.FileSize,
		FileDate:     result.FileDate,
		Etag:         downloadResult.Etag,
		LastModified: downloadResult.LastModified,
		ProcessedAt:  time.Now(),
	})

	return err
}

func processBcvFile(ctx context.Context, service *bcv.Service, link string, force bool) error {
	repo := NewRepository(ctx)

	var bcvFile *model.BcvFiles

	if !force {
		file, err := repo.Get(link)
		if err != nil {
			return err
		}

		bcvFile = file
	}

	downloadResult := service.Download(bcvFile, link)
	if downloadResult.Error != nil {
		return downloadResult.Error
	}

	return processResult(ctx, downloadResult)
}
