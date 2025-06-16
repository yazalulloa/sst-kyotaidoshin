package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yaz/kyo-repo/internal/util"
	"io"
	"log"
	"net/http"
	"time"
)

const siteVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

type SiteVerifyResponse struct {
	Success     bool      `json:"success"`
	Score       float64   `json:"score"`
	Action      string    `json:"action"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

func CheckRecaptcha(action RecaptchaAction, secret, response string) error {
	req, err := http.NewRequest(http.MethodPost, siteVerifyURL, nil)
	if err != nil {
		return err
	}

	// Add necessary request parameters.
	q := req.URL.Query()
	q.Add("secret", secret)
	q.Add("response", response)
	req.URL.RawQuery = q.Encode()

	// Make request
	resp, err := util.GetHttpClient().Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing body: ", err)
		}
	}(resp.Body)

	// Decode response.
	var body SiteVerifyResponse
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}

	// Check recaptcha verification success.
	if !body.Success {
		return errors.New(fmt.Sprintf("unsuccessful recaptcha verify request %s", body.ErrorCodes))
	}

	// Check response score.
	if body.Score < 0.5 {
		return errors.New(fmt.Sprintf("recaptcha score is too low: %s", util.FormatFloat64(body.Score)))
		//return errors.New("lower received score than expected")
	}

	// Check response action.
	if !action.Is(body.Action) {
		return errors.New("mismatched recaptcha action")
	}

	return nil
}

type RecaptchaAction string

const (
	ApartmentsUpsertRecaptchaAction       RecaptchaAction = "apartments_upsert"
	ApartmentsDeleteRecaptchaAction       RecaptchaAction = "apartments_delete"
	ApartmentsUploadBackupRecaptchaAction RecaptchaAction = "apartments_upload_backup"

	BuildingsUpsertRecaptchaAction       RecaptchaAction = "buildings_upsert"
	BuildingsDeleteRecaptchaAction       RecaptchaAction = "buildings_delete"
	BuildingsUploadBackupRecaptchaAction RecaptchaAction = "buildings_upload_backup"

	DebtsUpsertRecaptchaAction RecaptchaAction = "debts_upsert"

	ExpensesUpsertRecaptchaAction RecaptchaAction = "expenses_upsert"
	ExpensesDeleteRecaptchaAction RecaptchaAction = "expenses_delete"

	ExtraChargesUpsertRecaptchaAction RecaptchaAction = "extra_charges_upsert"
	ExtraChargesDeleteRecaptchaAction RecaptchaAction = "extra_charges_delete"

	PermissionsInsertAllRecaptchaAction RecaptchaAction = "permissions_insert_all"
	PermissionsDeleteRecaptchaAction    RecaptchaAction = "permissions_delete"

	RatesDeleteRecaptchaAction RecaptchaAction = "rates_delete"

	ReserveFundsUpsertRecaptchaAction RecaptchaAction = "reserve_funds_upsert"
	ReserveFundsDeleteRecaptchaAction RecaptchaAction = "reserve_funds_delete"

	RolesUpsertRecaptchaAction RecaptchaAction = "roles_upsert"
	RolesDeleteRecaptchaAction RecaptchaAction = "roles_delete"

	TelegramSetWebhookRecaptchaAction    RecaptchaAction = "telegram_set_webhook"
	TelegramDeleteWebhookRecaptchaAction RecaptchaAction = "telegram_delete_webhook"

	UsersDeleteRecaptchaAction  RecaptchaAction = "users_delete"
	UserRolesSetRecaptchaAction RecaptchaAction = "user_roles_set"

	ReceiptsCreateRecaptchaAction       RecaptchaAction = "receipts_create"
	ReceiptsDeleteRecaptchaAction       RecaptchaAction = "receipts_delete"
	ReceiptsUpdateRecaptchaAction       RecaptchaAction = "receipts_update"
	ReceiptsDuplicateRecaptchaAction    RecaptchaAction = "receipts_duplicate"
	ReceiptsParseFileRecaptchaAction    RecaptchaAction = "receipts_parse_file"
	ReceiptsUploadBackupRecaptchaAction RecaptchaAction = "receipts_upload_backup"
	ReceiptsDeletePdfsRecaptchaAction   RecaptchaAction = "receipts_delete_pdfs"
	ReceiptsSendAptsRecaptchaAction     RecaptchaAction = "receipts_send_apts"

	BcvBucketDeleteRecaptchaAction  RecaptchaAction = "bcv_bucket_delete"
	BcvBucketProcessRecaptchaAction RecaptchaAction = "bcv_bucket_process"
)

func (receiver RecaptchaAction) Name() string {
	return string(receiver)
}

func (receiver RecaptchaAction) Is(str string) bool {
	return receiver.Name() == str
}
