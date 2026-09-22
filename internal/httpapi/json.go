// Package httpapi wires the HTTP surface — one handler per Node route,
// same paths, same status codes, same JSON shapes — on top of the
// internal/service layer.
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/felpsalvs/pdv-starter/internal/service"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("erro ao serializar resposta: %v", err)
	}
}

func writeErrorMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeError maps an error the same way handleError() did in every Node
// router: a *service.ValidationError becomes 400 with its own message,
// anything else is logged and reported as a generic 500.
func writeError(w http.ResponseWriter, err error) {
	if ve, ok := err.(*service.ValidationError); ok {
		writeErrorMessage(w, http.StatusBadRequest, ve.Message)
		return
	}
	log.Println(err)
	writeErrorMessage(w, http.StatusInternalServerError, "Erro interno do servidor.")
}

func pathID(r *http.Request) (int64, bool) {
	return parseID(r.PathValue("id"))
}

func parseID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// flexNumber decodes a JSON number OR a numeric string into a float64,
// matching the JS backend's habit of doing Number(req.body.x) on every
// numeric input regardless of whether the client sent a string or a
// number. A missing/null field leaves Present false.
type flexNumber struct {
	Value   float64
	Present bool
	Valid   bool // false if present but not a finite number (mirrors Number.isFinite check failing)
}

func (n *flexNumber) UnmarshalJSON(data []byte) error {
	n.Present = true
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case float64:
		n.Value = v
		n.Valid = true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			n.Valid = false
			return nil
		}
		n.Value = f
		n.Valid = true
	case nil:
		n.Present = false
		n.Valid = false
	default:
		n.Valid = false
	}
	return nil
}
