package main

import (
	"context"
	"fmt"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"kyotaidoshin/email_h"
	"kyotaidoshin/receiptPdf"
	"kyotaidoshin/receipts"
	"kyotaidoshin/util"
	"log"
	"strings"
	"sync"
	"time"
)

func sendPdfs(ctx context.Context, event receiptPdf.QueueEvent) error {
	altEmailsRecipient, err := resource.Get("AltEmailsRecipient", "value")
	if err != nil {
		return fmt.Errorf("altEmailsRecipient: %v", err)
	}

	altRecipient := altEmailsRecipient.(string)

	receipt, err := receipts.CalculateReceipt(event.BuildingId, event.ReceiptId)
	if err != nil {
		return err
	}

	progressUpdate, err := receiptPdf.GetProgress(ctx, event.ProgressId)
	if err != nil {
		return err
	}

	// TODO TTL
	//defer func(ctx context.Context, objectKey string) {
	//	err := receiptPdf.DeleteProgress(ctx, objectKey)
	//	if err != nil {
	//		log.Printf("Error deleting progress: %s %v", objectKey, err)
	//	}
	//}(ctx, event.ProgressId)

	progressUpdate.Building = receipt.Building.Name
	progressUpdate.Month = receipt.MonthStr
	progressUpdate.Date = receipt.Receipt.Date.Format(time.DateOnly)

	err = receiptPdf.PutProgress(ctx, *progressUpdate)
	if err != nil {
		return err
	}

	parts, err := receipts.GetParts(receipt, ctx, true, &receipts.DownloadKeys{
		Parts:  event.Apartments,
		AllApt: len(event.Apartments) == 0,
	})
	if err != nil {
		return err
	}

	progressUpdate.Size = len(parts)
	progressUpdate.Counter = 0

	err = receiptPdf.PutProgress(ctx, *progressUpdate)
	if err != nil {
		return err
	}

	log.Printf("Parts %d", len(parts))

	var wg sync.WaitGroup
	messages := make([]*email_h.MsgWithCallBack, len(parts))
	wg.Add(len(parts))
	errChan := make(chan error, len(parts))

	for i, part := range parts {
		go func() {
			defer wg.Done()

			var emails []string
			if altRecipient == "" {
				emails = strings.Split(part.Apt.Emails, ",")
				log.Printf("Emails %v", emails)
			} else {
				emails = []string{altRecipient}
			}

			emails = []string{altRecipient}

			req := receiptPdf.SendPdfRequest{
				Emails:        emails,
				MonthStr:      receipt.MonthStr,
				Year:          receipt.Receipt.Year,
				BuildingName:  receipt.Building.Name,
				AptNumber:     part.Apt.Number,
				SubjectPrefix: "",
				Text:          "",
				ObjectKey:     part.ObjectKey,
				EmailKey:      receipt.Building.EmailConfig,
			}

			msg, err := receiptPdf.BuildMsg(ctx, req)
			if err != nil {
				errChan <- err
				return
			}
			messages[i] = &email_h.MsgWithCallBack{
				Msg: msg,
				Callback: func() {

					log.Printf("Sent %s", part.Apt.Number)
					progressUpdate.Counter++
					progressUpdate.Apt = part.Apt.Number
					progressUpdate.Name = part.Apt.Name
					progressUpdate.To = part.Apt.Emails
					err := receiptPdf.PutProgress(ctx, *progressUpdate)
					if err != nil {
						log.Printf("Error updating progress: %v", err)
					}

				},
			}
		}()
	}

	wg.Wait()
	close(errChan)

	err = util.HasErrors(errChan)
	if err != nil {
		return err
	}

	log.Printf("Sending %d", len(parts))
	from, err := email_h.GetFromEmail(receipt.Building.EmailConfig)
	if err != nil {
		return err
	}

	progressUpdate.From = from

	err = email_h.SendEmail(ctx, receipt.Building.EmailConfig, messages)
	if err != nil {
		log.Printf("Error sending email: %v", err)
		return err
	}

	log.Printf("Sent %d", len(parts))
	_, err = receipts.UpdateLastSent(event.ReceiptId)
	if err != nil {
		return err
	}

	progressUpdate.Finished = true
	err = receiptPdf.PutProgress(ctx, *progressUpdate)
	if err != nil {
		return err
	}

	return nil
}
