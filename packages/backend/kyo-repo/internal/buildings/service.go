package buildings

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/email_h"
	"github.com/yaz/kyo-repo/internal/extraCharges"
	"github.com/yaz/kyo-repo/internal/reserveFunds"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"strings"
	"sync"
)

type Service struct {
	repo Repository
}

func NewService(ctx context.Context) Service {
	return Service{
		repo: NewRepository(ctx),
	}
}

func (service Service) getTableResponse(requestQuery RequestQuery) (*TableResponse, error) {
	var rateTableResponse TableResponse

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		array, err := service.repo.selectList(requestQuery)
		if err != nil {
			errorChan <- err
			return
		}

		results := make([]Item, len(array))
		for i, item := range array {
			//log.Printf("ID %d aptCount %d\n", item.ID, *item.AptCount)

			results[i] = Item{
				Key:       *util.Encode(item.ID),
				CardId:    "buildings-" + uuid.NewString(),
				Item:      item.Buildings,
				AptCount:  item.AptCount,
				CreatedAt: item.CreatedAt.UnixMilli(),
			}

		}
		rateTableResponse.Results = results
	}()

	go func() {
		defer wg.Done()
		totalCount, err := service.repo.getTotalCount()
		if err != nil {
			errorChan <- err
			return
		}
		rateTableResponse.Counters.TotalCount = totalCount
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	return &rateTableResponse, nil
}

func (service Service) deleteAndReturnCounters(id string) (*Counters, error) {

	var wg sync.WaitGroup
	workers := 3
	wg.Add(workers)
	errorChan := make(chan error, workers)

	go func() {
		defer wg.Done()
		_, err := service.repo.deleteById(id)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		_, err := reserveFunds.NewRepository(service.repo.ctx).DeleteByBuilding(id)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		_, err := extraCharges.NewRepository(service.repo.ctx).DeleteByBuilding(id)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	totalCount, err := service.repo.getTotalCount()
	if err != nil {
		return nil, err
	}

	counters := Counters{}
	counters.TotalCount = totalCount
	return &counters, nil
}

func (service Service) Backup() (string, error) {

	lastId := ""

	selectListDtos := func() ([]BuildingRecord, error) {
		list, err := service.repo.selectRecords(lastId, 2)
		if err != nil {
			return nil, err
		}

		log.Printf("Backup buildings count %d\n", len(list))

		dtos := make([]BuildingRecord, len(list))

		for i, item := range list {
			reserveFundArray := make([]reserveFundDto, len(item.ReserveFunds))

			for j, reserveFund := range item.ReserveFunds {
				reserveFundArray[j] = reserveFundDto{
					BuildingID:    reserveFund.BuildingID,
					Name:          reserveFund.Name,
					Fund:          reserveFund.Fund,
					Expense:       reserveFund.Expense,
					Pay:           reserveFund.Pay,
					Active:        reserveFund.Active,
					Type:          reserveFund.Type,
					ExpenseType:   reserveFund.ExpenseType,
					AddToExpenses: reserveFund.AddToExpenses,
				}
			}

			extraChargesArray := make([]extraCharges.ExtraChargeDto, len(item.ExtraCharges))

			for j, extraCharge := range item.ExtraCharges {
				apts := strings.Split(extraCharge.Apartments, ",")

				aptsDto := make([]extraCharges.AptDto, len(apts))
				for k, apt := range apts {
					aptsDto[k] = extraCharges.AptDto{
						Number: apt,
					}
				}

				extraChargesArray[j] = extraCharges.ExtraChargeDto{
					BuildingID:      extraCharge.BuildingID,
					ParentReference: extraCharge.ParentReference,
					Type:            extraCharge.Type,
					Description:     extraCharge.Description,
					Amount:          extraCharge.Amount,
					Currency:        extraCharge.Currency,
					Active:          extraCharge.Active,
					Apartments:      aptsDto,
				}
			}

			dtos[i] = BuildingRecord{
				Building: buildingDto{
					Id:                          item.Buildings.ID,
					Name:                        item.Buildings.Name,
					Rif:                         item.Buildings.Rif,
					MainCurrency:                item.Buildings.MainCurrency,
					DebtCurrency:                item.Buildings.DebtCurrency,
					CurrenciesToShowAmountToPay: strings.Split(item.Buildings.CurrenciesToShowAmountToPay, ","),
					FixedPay:                    item.Buildings.FixedPay,
					FixedPayAmount:              item.Buildings.FixedPayAmount,
					RoundUpPayments:             item.Buildings.RoundUpPayments,
				},
				ReserveFunds: reserveFundArray,
				ExtraCharges: extraChargesArray,
			}

			if i == len(list)-1 {
				lastId = item.Buildings.ID
			}
		}

		return dtos, nil
	}

	return api.Backup(selectListDtos)
}

func (service Service) ProcessDecoder(decoder *json.Decoder) (int64, error) {
	var dto []BuildingRecord
	err := decoder.Decode(&dto)
	if err != nil {
		log.Printf("Error decoding json: %s", err)
		return 0, err
	}

	buildings := make([]model.Buildings, len(dto))
	var reserveFundArray []model.ReserveFunds
	var extraChargeArray []model.ExtraCharges

	configs, err := email_h.GetConfigs()
	if err != nil {
		return 0, err
	}

	getFirst := func() string {
		for key := range configs {
			return key
		}
		return ""
	}

	for i, record := range dto {
		buildings[i] = model.Buildings{
			ID:                          record.Building.Id,
			Name:                        record.Building.Name,
			Rif:                         record.Building.Rif,
			MainCurrency:                record.Building.MainCurrency,
			DebtCurrency:                record.Building.DebtCurrency,
			CurrenciesToShowAmountToPay: strings.Join(record.Building.CurrenciesToShowAmountToPay, ","),
			FixedPay:                    record.Building.FixedPay,
			FixedPayAmount:              record.Building.FixedPayAmount,
			RoundUpPayments:             record.Building.RoundUpPayments,
			EmailConfig:                 getFirst(),
		}

		for _, reserveFund := range record.ReserveFunds {
			reserveFundArray = append(reserveFundArray, model.ReserveFunds{
				BuildingID:    reserveFund.BuildingID,
				Name:          reserveFund.Name,
				Fund:          reserveFund.Fund,
				Expense:       reserveFund.Expense,
				Pay:           reserveFund.Pay,
				Active:        reserveFund.Active,
				Type:          reserveFund.Type,
				ExpenseType:   reserveFund.ExpenseType,
				AddToExpenses: reserveFund.AddToExpenses,
			})
		}

		for _, extraCharge := range record.ExtraCharges {
			var builder strings.Builder
			for idx, apt := range extraCharge.Apartments {
				builder.WriteString(apt.Number)
				if idx < len(extraCharge.Apartments)-1 {
					builder.WriteString(",")
				}
			}

			extraChargeArray = append(extraChargeArray, model.ExtraCharges{
				BuildingID:      extraCharge.BuildingID,
				ParentReference: extraCharge.ParentReference,
				Type:            extraCharge.Type,
				Description:     extraCharge.Description,
				Amount:          extraCharge.Amount,
				Currency:        extraCharge.Currency,
				Active:          extraCharge.Active,
				Apartments:      builder.String(),
			})
		}
	}

	rowsAffected, err := service.repo.insertBackup(buildings)
	if err != nil {
		return 0, err
	}

	_, err = reserveFunds.NewRepository(service.repo.ctx).InsertBackup(reserveFundArray)
	if err != nil {
		return 0, err
	}

	_, err = extraCharges.NewRepository(service.repo.ctx).InsertBulk(extraChargeArray)
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
