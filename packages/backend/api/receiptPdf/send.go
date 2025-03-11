package receiptPdf

import (
	"aws_h"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/wneessen/go-mail"
	"kyotaidoshin/util"
	"strings"
)

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
			return nil, err
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

	return message, nil
}
