package buildings

import (
	"github.com/google/uuid"
	"kyotaidoshin/api"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/reserveFunds"
	"kyotaidoshin/util"
	"log"
	"strings"
	"sync"
)

func getTableResponse(requestQuery RequestQuery) (*TableResponse, error) {
	var rateTableResponse TableResponse

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		array, err := selectList(requestQuery)
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
		totalCount, err := getTotalCount()
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

func deleteAndReturnCounters(id string) (*Counters, error) {

	var wg sync.WaitGroup
	workers := 3
	wg.Add(workers)
	errorChan := make(chan error, workers)

	go func() {
		defer wg.Done()
		_, err := deleteById(id)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		_, err := reserveFunds.DeleteByBuilding(id)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		_, err := extraCharges.DeleteByBuilding(id)
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

	totalCount, err := getTotalCount()
	if err != nil {
		return nil, err
	}

	counters := Counters{}
	counters.TotalCount = totalCount
	return &counters, nil
}

func Backup() (string, error) {

	lastId := ""

	selectListDtos := func() ([]BuildingRecord, error) {
		list, err := selectRecords(lastId, 2)
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

	return api.Backup(api.BACKUP_BUILDINGS_FILE, selectListDtos)
}
