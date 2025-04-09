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

type Holder struct {
	ctx   context.Context
	event receiptPdf.QueueEvent
}

func (holder *Holder) update(pf func(update *receiptPdf.ProgressUpdate) error) (bool, error) {
	progressUpdate, err := receiptPdf.GetProgress(holder.ctx, holder.event.ProgressId)
	if err != nil {
		return false, err
	}

	if progressUpdate.Cancelled {
		return false, nil
	}

	err = pf(progressUpdate)
	if err != nil {
		progressUpdate.ErrMsg = err.Error()
		progressUpdate.Finished = true
	}

	err = receiptPdf.PutProgress(holder.ctx, progressUpdate)
	if err != nil {
		return false, err
	}

	return true, nil

}

func (holder *Holder) _sendPdfs() error {
	altEmailsRecipient, err := resource.Get("AltEmailsRecipient", "value")
	if err != nil {
		return fmt.Errorf("altEmailsRecipient: %v", err)
	}

	altRecipient := altEmailsRecipient.(string)

	receipt, err := receipts.CalculateReceipt(holder.event.BuildingId, holder.event.ReceiptId, holder.event.KeyStr)
	if err != nil {
		return err
	}

	shouldContinue, err := holder.update(func(update *receiptPdf.ProgressUpdate) error {
		update.Building = receipt.Building.Name
		update.Month = receipt.MonthStr
		update.Date = receipt.Receipt.Date.Format(time.DateOnly)
		return nil
	})

	if err != nil {
		return err
	}

	if !shouldContinue {
		log.Printf("Cancelled")
		return nil
	}

	parts, err := receipts.GetParts(receipt, holder.ctx, true, &receipts.DownloadKeys{
		Parts:  holder.event.Apartments,
		AllApt: len(holder.event.Apartments) == 0,
	})

	if err != nil {
		return err
	}

	shouldContinue, err = holder.update(func(update *receiptPdf.ProgressUpdate) error {
		update.Size = len(parts)
		update.Counter = 0
		return nil
	})

	if err != nil {
		return err
	}

	if !shouldContinue {
		log.Printf("Cancelled")
		return nil
	}

	log.Printf("Parts %d", len(parts))

	var wg sync.WaitGroup
	messages := make([]*email_h.MsgWithCallBack, len(parts))
	wg.Add(len(parts))
	errChan := make(chan error, len(parts))

	sentMsgs := 0

	for i, part := range parts {
		go func() {
			defer wg.Done()

			var emails []string
			if altRecipient == "" {
				emails = strings.Split(part.Apt.Emails, ",")
			} else {
				emails = []string{altRecipient}
			}

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

			msg, err := receiptPdf.BuildMsg(holder.ctx, req)
			if err != nil {
				errChan <- err
				return
			}
			messages[i] = &email_h.MsgWithCallBack{
				Msg: msg,
				Callback: func() {

					sentMsgs++

					shouldContinue, err = holder.update(func(update *receiptPdf.ProgressUpdate) error {
						update.Counter++
						update.Apt = part.Apt.Number
						update.Name = part.Apt.Name
						update.To = part.Apt.Emails
						return nil
					})

					if err != nil {
						log.Printf("Error updating progress: %v", err)
					}

				},
				ShouldContinue: func() bool {
					return shouldContinue
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

	shouldContinue, err = holder.update(func(update *receiptPdf.ProgressUpdate) error {
		update.From = from
		return nil
	})

	if err != nil {
		return err
	}

	if !shouldContinue {
		log.Printf("Cancelled")
		return nil
	}

	err = email_h.SendEmail(holder.ctx, receipt.Building.EmailConfig, messages)
	if err != nil {
		log.Printf("Error sending email: %v", err)
		return err
	}

	if !shouldContinue {
		log.Printf("Cancelled before last sent")
		return nil
	}

	log.Printf("Sent %d", sentMsgs)
	_, err = receipts.UpdateLastSent(holder.event.ReceiptId)
	if err != nil {
		return err
	}

	_, err = holder.update(func(update *receiptPdf.ProgressUpdate) error {
		update.Finished = true
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (holder *Holder) sendPdfs() error {
	err := holder._sendPdfs()
	if err != nil {
		log.Printf("Error sending PDFs: %v", err)
		_, updateErr := holder.update(func(update *receiptPdf.ProgressUpdate) error {
			update.ErrMsg = err.Error()
			update.Finished = true
			return nil
		})
		if updateErr != nil {
			log.Printf("Error updating progress: %v", err)
			return updateErr
		}
	}

	return nil
}
