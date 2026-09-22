package httpapi

import "net/http"

// NewRouter mirrors server.js's route table 1:1: the same paths mounted
// under /api/*, plus static (the embedded frontend build) for everything
// else.
func NewRouter(h *Handlers, static http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/categories", h.listCategories)
	mux.HandleFunc("POST /api/categories", h.createCategory)
	mux.HandleFunc("PUT /api/categories/{id}", h.updateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", h.deleteCategory)

	mux.HandleFunc("GET /api/products", h.listProducts)
	mux.HandleFunc("POST /api/products", h.createProduct)
	mux.HandleFunc("PUT /api/products/{id}", h.updateProduct)
	mux.HandleFunc("PATCH /api/products/{id}/availability", h.setProductAvailability)
	mux.HandleFunc("DELETE /api/products/{id}", h.deleteProduct)

	mux.HandleFunc("POST /api/orders", h.createOrder)
	mux.HandleFunc("GET /api/orders/today", h.listOrdersToday)
	mux.HandleFunc("GET /api/orders/{id}", h.getOrder)
	mux.HandleFunc("POST /api/orders/{id}/pay", h.payOrder)
	mux.HandleFunc("POST /api/orders/{id}/cancel", h.cancelOrder)
	mux.HandleFunc("POST /api/orders/{id}/reprint", h.reprintOrder)

	mux.HandleFunc("GET /api/cash-register/current", h.getCurrentRegister)
	mux.HandleFunc("POST /api/cash-register/open", h.openRegister)
	mux.HandleFunc("POST /api/cash-register/movement", h.registerCashMovement)
	mux.HandleFunc("POST /api/cash-register/close", h.closeRegister)
	mux.HandleFunc("GET /api/cash-register/closures", h.listClosures)

	mux.HandleFunc("GET /api/report/day", h.getDayReport)

	mux.Handle("/", static)

	return mux
}
