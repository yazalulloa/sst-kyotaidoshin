package isr

import (
	"context"
	"kyotaidoshin/api"
	"net/http"
)

func Routes(holder *api.RouterHolder) {

	holder.GET("/api/rates/currencies.html", renderObj(GetRatesCurrencies))
	holder.GET("/api/receipts/buildings.html", renderObj(GetReceiptsBuildings))
	holder.GET("/api/receipts/years.html", renderObj(GetReceiptsYears))
	holder.GET("/api/apartments/buildings.html", renderObj(GetApartmentsBuildings))
}

func renderObj(dataFunc func(ctx context.Context) ([]byte, error)) func(w http.ResponseWriter, r *http.Request) {

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
