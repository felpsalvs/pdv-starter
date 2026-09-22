package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/felpsalvs/pdv-starter/internal/service"
)

type orderItemBody struct {
	ProductID flexNumber `json:"productId"`
	Quantity  flexNumber `json:"quantity"`
	Note      string     `json:"note"`
}

type paymentBody struct {
	PaymentMethod  string     `json:"paymentMethod"`
	AmountReceived flexNumber `json:"amountReceived"`
}

func toItemInputs(items []orderItemBody) []service.ItemInput {
	result := make([]service.ItemInput, len(items))
	for i, it := range items {
		productID := int64(0)
		if it.ProductID.Valid {
			productID = int64(it.ProductID.Value)
		}
		quantity := int64(0)
		if it.Quantity.Valid {
			quantity = int64(it.Quantity.Value)
		}
		result[i] = service.ItemInput{ProductID: productID, Quantity: quantity, Note: it.Note}
	}
	return result
}

func toPaymentInput(p *paymentBody) *service.PaymentInput {
	if p == nil {
		return nil
	}
	return &service.PaymentInput{
		HasPaymentMethod:  p.PaymentMethod != "",
		PaymentMethod:     p.PaymentMethod,
		HasAmountReceived: p.AmountReceived.Present && p.AmountReceived.Valid,
		AmountReceived:    p.AmountReceived.Value,
	}
}

func (h *Handlers) createOrder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source    string          `json:"source"`
		Reference string          `json:"reference"`
		Items     []orderItemBody `json:"items"`
		Note      string          `json:"note"`
		Payment   *paymentBody    `json:"payment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "O pedido precisa ter pelo menos um item.")
		return
	}

	result, err := h.orders.CreateOrderWithPrinting(service.CreateOrderInput{
		Source:    body.Source,
		Reference: body.Reference,
		Items:     toItemInputs(body.Items),
		Note:      body.Note,
		Payment:   toPaymentInput(body.Payment),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handlers) listOrdersToday(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	orders, err := h.orders.ListForDay(q.Get("date"), q.Get("status"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (h *Handlers) getOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusNotFound, "Pedido não encontrado.")
		return
	}
	order, err := h.orders.LoadOrder(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if order == nil {
		writeErrorMessage(w, http.StatusNotFound, "Pedido não encontrado.")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *Handlers) payOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusBadRequest, "Pedido não encontrado.")
		return
	}
	var body paymentBody
	_ = json.NewDecoder(r.Body).Decode(&body)

	result, err := h.orders.RegisterPayment(id, toPaymentInput(&body))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) cancelOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusBadRequest, "Pedido não encontrado.")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	order, err := h.orders.CancelOrder(id, body.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *Handlers) reprintOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusBadRequest, "Pedido não encontrado.")
		return
	}
	var body struct {
		Document string `json:"document"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	result, err := h.orders.Reprint(id, body.Document)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
