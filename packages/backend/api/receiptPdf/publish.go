package receiptPdf

import (
	"aws_h"
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"log"
)

type Type string

const (
	BuildingChanges Type = "BuildingChanges"
	ReceiptChanges  Type = "ReceiptChanges"
	SendPdfs        Type = "SendPdfs"
)

type QueueEvent struct {
	Type       Type     `json:"type"`
	BuildingId string   `json:"buildingId"`
	ReceiptId  string   `json:"receiptId"`
	ProgressId string   `json:"progressId"`
	Apartments []string `json:"apartments"`
	Subject    string   `json:"subject"`
	Message    string   `json:"message"`
}

func (receiver QueueEvent) IsChanges() bool {
	return receiver.Type == BuildingChanges || receiver.Type == ReceiptChanges
}

const pdfChangesMessageGroupId = "PdfChanges"
const sendPdfsMessageGroupId = "SendPdfs"

func PublishBuilding(ctx context.Context, buildingId string) {
	event := QueueEvent{
		Type:       BuildingChanges,
		BuildingId: buildingId,
		ReceiptId:  "",
	}

	err := publishEvent(ctx, event, pdfChangesMessageGroupId, nil)
	if err != nil {
		log.Printf("Error publishing building: %v", err)
	}
}

func PublishReceipt(ctx context.Context, buildingId string, receiptId string) {
	event := QueueEvent{
		Type:       ReceiptChanges,
		BuildingId: buildingId,
		ReceiptId:  receiptId,
	}

	err := publishEvent(ctx, event, pdfChangesMessageGroupId, nil)
	if err != nil {
		log.Printf("Error publishing receipt: %v", err)
	}
}

func publishEvent(ctx context.Context, event QueueEvent, messageGroupId string, messageDeduplicationId *string) error {

	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	queue, err := resource.Get("ReceiptPdfQueue", "url")
	if err != nil {
		return err
	}

	client, err := aws_h.GetSqsClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:               aws.String(queue.(string)),
		MessageBody:            aws.String(string(bytes)),
		MessageGroupId:         aws.String(messageGroupId),
		MessageDeduplicationId: messageDeduplicationId,
	})

	if err != nil {
		return err
	}

	return nil
}

type PublishSendPdfsRequest struct {
	Ctx        context.Context
	BuildingId string
	ReceiptId  string
	CardId     string
	Apartments []string
	Subject    string
	Message    string
}

func PublishSendPdfs(req PublishSendPdfsRequest) (string, error) {
	deduplicationId := uuid.NewString()
	event := QueueEvent{
		Type:       SendPdfs,
		BuildingId: req.BuildingId,
		ReceiptId:  req.ReceiptId,
		ProgressId: deduplicationId,
		Apartments: req.Apartments,
		Subject:    req.Subject,
		Message:    req.Message,
	}

	update := ProgressUpdate{ObjectKey: deduplicationId, CardId: "sent-" + req.CardId}
	err := PutProgress(req.Ctx, &update)
	if err != nil {
		log.Printf("Error putting progress: %v", err)
		return "", err
	}

	err = publishEvent(req.Ctx, event, sendPdfsMessageGroupId+deduplicationId, &deduplicationId)
	if err != nil {
		log.Printf("Error publishing send pdfs: %v", err)
		return "", err
	}

	return deduplicationId, nil
}
