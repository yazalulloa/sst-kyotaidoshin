package bcv_files

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yaz/kyo-repo/internal/bcv"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/rates"
	"github.com/yaz/kyo-repo/internal/telegram"
	"github.com/yaz/kyo-repo/internal/util"
)

type Service struct {
	ctx          context.Context
	bcvService   *bcv.Service
	bcvFilesRepo Repository
	ratesSer     rates.Service
}

func NewService(ctx context.Context) *Service {
	bcvService, err := bcv.NewService(ctx)
	if err != nil {
		log.Fatalf("Error creating BCV service: %s", err)
	}

	return &Service{
		ctx:          ctx,
		bcvService:   bcvService,
		bcvFilesRepo: NewRepository(ctx),
		ratesSer:     rates.NewService(ctx),
	}
}

func (ser Service) processAllFiles() error {

	links, err := ser.bcvService.FileLinks(false)

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

		go func(link string, pos int) {
			defer wg.Done()

			_, pErr := ser.processBcvFile(link, true)

			if pErr != nil {
				errorChan <- pErr
				return
			}
		}(link, pos)
	}

	wg.Wait()
	close(errorChan)

	return util.HasErrors(errorChan)
}

func (ser Service) processResult(downloadResult bcv.DownloadResult) (*bcv.Result, error) {
	info := bcv.ParsingInfo{
		BucketKey:  downloadResult.Link,
		FilePath:   downloadResult.FilePath,
		ProcessAll: true,
		Ctx:        ser.ctx,
	}

	result, err := info.Parse()
	if err != nil {
		return nil, err
	}

	var toInsert []*model.Rates
	for i := 0; i < len(result.Rates); i++ {
		array := result.Rates[i]
		for _, lhs := range array {
			var rhs *model.Rates

			for j := i + 1; j < len(result.Rates); j++ {
				secondArray := result.Rates[j]
				for _, v := range secondArray {
					if lhs.FromCurrency == v.FromCurrency {
						rhs = v
						break
					}
				}

				if rhs != nil {
					break
				}
			}

			rates.CalculateTrend(lhs, rhs)

			toInsert = append(toInsert, lhs)
		}
	}

	if len(toInsert) == 0 {
		return nil, fmt.Errorf("no rates to insert from file %s", info.BucketKey)
	}

	log.Printf("Inserting %d rates from file %s", len(toInsert), info.BucketKey)

	ratesInserted, err := rates.NewRepository(info.Ctx).Insert(toInsert)

	if err != nil {
		return nil, fmt.Errorf("error inserting rates from file %s: %s", info.BucketKey, err)
	}

	result.Inserted += ratesInserted

	log.Printf("Processed %s rates parsed: %d inserted: %d", downloadResult.Link, result.Parsed, result.Inserted)

	_, err = ser.bcvFilesRepo.Insert(model.BcvFiles{
		Link:         &downloadResult.Link,
		RateCount:    int64(result.Parsed),
		SheetCount:   int32(result.NumOfSheets),
		FileSize:     downloadResult.FileSize,
		FileDate:     result.FileDate,
		Etag:         downloadResult.Etag,
		LastModified: downloadResult.LastModified,
		ProcessedAt:  time.Now(),
	})

	return result, err
}

func (ser Service) processBcvFile(link string, force bool) (*bcv.Result, error) {

	var bcvFile *model.BcvFiles

	if !force {
		file, err := ser.bcvFilesRepo.Get(link)
		if err != nil {
			return nil, err
		}

		bcvFile = file
	}

	downloadResult := ser.bcvService.Download(bcvFile, link)
	if downloadResult.Error != nil {
		return nil, downloadResult.Error
	}

	if downloadResult.FilePath == "" {
		log.Printf("File %s has not changed, skipping", link)
		return &bcv.Result{}, nil
	}

	return ser.processResult(downloadResult)
}

func (ser Service) BcvJob() error {
	links, err := ser.bcvService.FileLinks(true)

	if err != nil {
		return err
	}

	length := len(links)
	if length == 0 {
		log.Println("No files to check")
		return nil
	}

	if length > 1 {
		return fmt.Errorf("too many files to check (%d)", length)
	}

	link := links[0]

	result, err := ser.processBcvFile(link, false)
	if err != nil {
		return err
	}

	if result.Inserted > 0 {
		for _, rate := range result.Rates[0] {
			if rate.FromCurrency == "USD" || rate.FromCurrency == "EUR" {
				log.Printf("Sending %s rate: %f", rate.FromCurrency, rate.Rate)
				telegram.SendRate(ser.ctx, *rate)
			}
		}

		return ser.ratesSer.UpdateStableTrend()
	}

	return nil
}
