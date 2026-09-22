package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/felpsalvs/pdv-starter/internal/store"
)

func (h *Handlers) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.ProductFilter{
		CategoryID:      q.Get("category"),
		AvailableToday:  q.Get("availableToday") == "1",
		IncludeInactive: q.Get("includeInactive") != "",
	}
	products, err := h.store.ListProducts(filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func categoryIDFromJSON(n *json.Number) *int64 {
	if n == nil {
		return nil
	}
	f, err := n.Float64()
	if err != nil {
		return nil
	}
	v := int64(f)
	return &v
}

func (h *Handlers) createProduct(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name       string       `json:"name"`
		Price      flexNumber   `json:"price"`
		CategoryID *json.Number `json:"categoryId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "Informe o nome do produto.")
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeErrorMessage(w, http.StatusBadRequest, "Informe o nome do produto.")
		return
	}
	if !body.Price.Valid || body.Price.Value <= 0 {
		writeErrorMessage(w, http.StatusBadRequest, "Informe um preço válido.")
		return
	}

	highest, err := h.store.HighestProductSortOrder()
	if err != nil {
		writeError(w, err)
		return
	}

	product, err := h.store.CreateProduct(name, body.Price.Value, categoryIDFromJSON(body.CategoryID), highest+1)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, product)
}

func (h *Handlers) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusNotFound, "Produto não encontrado.")
		return
	}
	product, err := h.store.GetProduct(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if product == nil {
		writeErrorMessage(w, http.StatusNotFound, "Produto não encontrado.")
		return
	}

	var body struct {
		Name       *string      `json:"name"`
		Price      flexNumber   `json:"price"`
		CategoryID *json.Number `json:"categoryId"`
		Active     *bool        `json:"active"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	newName := product.Name
	if body.Name != nil {
		newName = strings.TrimSpace(*body.Name)
	}
	newPrice := product.Price
	if body.Price.Present {
		newPrice = body.Price.Value
	}
	newCategory := product.CategoryID
	if body.CategoryID != nil {
		newCategory = categoryIDFromJSON(body.CategoryID)
	}
	newActive := product.Active
	if body.Active != nil {
		if *body.Active {
			newActive = 1
		} else {
			newActive = 0
		}
	}

	if newName == "" {
		writeErrorMessage(w, http.StatusBadRequest, "Informe o nome do produto.")
		return
	}
	if body.Price.Present && !body.Price.Valid {
		writeErrorMessage(w, http.StatusBadRequest, "Informe um preço válido.")
		return
	}
	if newPrice <= 0 {
		writeErrorMessage(w, http.StatusBadRequest, "Informe um preço válido.")
		return
	}

	updated, err := h.store.UpdateProduct(id, newName, newPrice, newCategory, newActive)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handlers) setProductAvailability(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusNotFound, "Produto não encontrado.")
		return
	}
	product, err := h.store.GetProduct(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if product == nil {
		writeErrorMessage(w, http.StatusNotFound, "Produto não encontrado.")
		return
	}

	var body struct {
		AvailableToday bool `json:"availableToday"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	available := int64(0)
	if body.AvailableToday {
		available = 1
	}
	updated, err := h.store.SetProductAvailability(id, available)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handlers) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusNotFound, "Produto não encontrado.")
		return
	}
	product, err := h.store.GetProduct(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if product == nil {
		writeErrorMessage(w, http.StatusNotFound, "Produto não encontrado.")
		return
	}
	if err := h.store.DeactivateProduct(id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
