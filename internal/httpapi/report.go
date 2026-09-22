package httpapi

import (
	"net/http"

	"github.com/felpsalvs/pdv-starter/internal/clock"
)

func (h *Handlers) getDayReport(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = clock.TodayLocal()
	}

	byPaymentMethod, err := h.store.ByPaymentMethod(date)
	if err != nil {
		writeError(w, err)
		return
	}
	topProducts, err := h.store.TopProducts(date)
	if err != nil {
		writeError(w, err)
		return
	}
	summary, err := h.store.DaySummary(date)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"date":            date,
		"summary":         summary,
		"byPaymentMethod": byPaymentMethod,
		"topProducts":     topProducts,
	})
}
