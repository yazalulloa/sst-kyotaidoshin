package receiptPdf

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/wneessen/go-mail"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/util"
)

type Attachment struct {
	FilePath string
	Name     string
}

type SendPdfRequest struct {
	Emails        []string
	MonthStr      string
	Year          int16
	BuildingName  string
	AptNumber     string
	SubjectPrefix string
	Text          string
	ObjectKey     string
	EmailKey      string
	Attachments   []Attachment
}

func BuildMsg(ctx context.Context, req SendPdfRequest) (*mail.Msg, error) {
	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		return nil, err
	}

	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		return nil, err
	}

	res, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(req.ObjectKey),
	})
	if err != nil {
		return nil, err
	}

	message := mail.NewMsg(mail.WithCharset(mail.CharsetUTF8))

	for _, email := range req.Emails {
		if err := message.AddTo(email); err != nil {
			return nil, fmt.Errorf("failed to add email %s: %v", email, err)
		}
	}

	prefixSub := strings.TrimSpace(req.SubjectPrefix)
	if prefixSub == "" {
		prefixSub = "AVISO DE COBRO"
	}

	message.Subject(prefixSub + fmt.Sprintf(" %s %d Adm. %s APT: %s", req.MonthStr, req.Year, req.BuildingName, req.AptNumber))

	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = "AVISO DE COBRO"
	}

	message.SetBodyString(mail.TypeTextPlain, text)

	err = message.AttachReader(req.AptNumber, res.Body,
		mail.WithFileName(fmt.Sprintf("%s.pdf", req.AptNumber)),
		mail.WithFileContentType("application/pdf"),
	)
	if err != nil {
		return nil, err
	}

	for _, attachment := range req.Attachments {
		f, err := os.Open(attachment.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open attachment %s: %w", attachment.Name, err)
		}
		err = message.AttachReader(attachment.Name, f)
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("failed to attach %s: %w", attachment.Name, err)
		}
		// f will be read during SendEmail; go-mail buffers internally so we can close after AttachReader
		_ = f.Close()
	}

	return message, nil
}
