package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Handlers) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.store.ListActiveCategories()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *Handlers) createCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "Informe o nome da categoria.")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeErrorMessage(w, http.StatusBadRequest, "Informe o nome da categoria.")
		return
	}

	highest, err := h.store.HighestCategorySortOrder()
	if err != nil {
		writeError(w, err)
		return
	}
	category, err := h.store.CreateCategory(name, highest+1)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

func (h *Handlers) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusNotFound, "Categoria não encontrada.")
		return
	}
	category, err := h.store.GetCategory(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if category == nil {
		writeErrorMessage(w, http.StatusNotFound, "Categoria não encontrada.")
		return
	}

	var body struct {
		Name      *string      `json:"name"`
		SortOrder *json.Number `json:"sortOrder"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	newName := category.Name
	if body.Name != nil {
		newName = strings.TrimSpace(*body.Name)
	}
	newSortOrder := category.SortOrder
	if body.SortOrder != nil {
		if f, err := body.SortOrder.Float64(); err == nil {
			newSortOrder = int64(f)
		}
	}

	if newName == "" {
		writeErrorMessage(w, http.StatusBadRequest, "Informe o nome da categoria.")
		return
	}

	updated, err := h.store.UpdateCategory(id, newName, newSortOrder)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handlers) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErrorMessage(w, http.StatusNotFound, "Categoria não encontrada.")
		return
	}
	category, err := h.store.GetCategory(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if category == nil {
		writeErrorMessage(w, http.StatusNotFound, "Categoria não encontrada.")
		return
	}
	if err := h.store.DeactivateCategory(id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
