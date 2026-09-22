package httpapi

import (
	"github.com/felpsalvs/pdv-starter/internal/service"
	"github.com/felpsalvs/pdv-starter/internal/store"
)

type Handlers struct {
	store        *store.Store
	orders       *service.OrdersService
	cashRegister *service.CashRegisterService
}

func NewHandlers(s *store.Store, orders *service.OrdersService, cashRegister *service.CashRegisterService) *Handlers {
	return &Handlers{store: s, orders: orders, cashRegister: cashRegister}
}
