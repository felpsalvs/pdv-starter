package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Handlers) getCurrentRegister(w http.ResponseWriter, r *http.Request) {
	summary, err := h.cashRegister.CurrentSummary()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handlers) openRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OpeningAmount flexNumber `json:"openingAmount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	register, err := h.cashRegister.Open(body.OpeningAmount.Value, body.OpeningAmount.Present && body.OpeningAmount.Valid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, register)
}

func (h *Handlers) registerCashMovement(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type   string     `json:"type"`
		Amount flexNumber `json:"amount"`
		Reason string     `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	summary, err := h.cashRegister.RegisterMovement(
		body.Type, body.Amount.Value, body.Amount.Present && body.Amount.Valid, strings.TrimSpace(body.Reason),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, summary)
}

func (h *Handlers) closeRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CountedAmount flexNumber `json:"countedAmount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	result, err := h.cashRegister.Close(body.CountedAmount.Value, body.CountedAmount.Present && body.CountedAmount.Valid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) listClosures(w http.ResponseWriter, r *http.Request) {
	closures, err := h.cashRegister.ListClosures()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, closures)
}
