package isr

import (
	"context"
	"github.com/yaz/kyo-repo/internal/api"
	"net/http"
)

func Routes(holder *api.RouterHolder) {

	holder.GET("/api/isr/rates/currencies.html", renderObj(ratesCurrenciesObjectKey))
	holder.GET("/api/isr/receipts/buildings.html", renderObj(receiptsBuildingsObjectKey))
	holder.GET("/api/isr/receipts/years.html", renderObj(receiptsYearsObjectKey))
	holder.GET("/api/isr/receipts/apartments.html", renderObj(receiptApartmentsObjectKey))
	holder.GET("/api/isr/apartments/buildings.html", renderObj(apartmentsBuildingsObjectKey))
}

func renderObj(objectKey string) func(w http.ResponseWriter, r *http.Request) {
	var dataFunc func(ctx context.Context) ([]byte, error)
	for _, obj := range isrObjects {
		if obj.objectKey == objectKey {
			dataFunc = obj.getObject
		}
	}

	if dataFunc == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Object not found", http.StatusInternalServerError)
			return
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		data, err := dataFunc(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "text/html")
		_, err = w.Write(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

}
