package service

import (
	"fmt"
	"strings"

	"github.com/felpsalvs/pdv-starter/internal/clock"
	"github.com/felpsalvs/pdv-starter/internal/printer"
	"github.com/felpsalvs/pdv-starter/internal/store"
)

var validSources = map[string]bool{"counter": true, "table": true}
var validPaymentMethods = map[string]bool{"cash": true, "pix": true, "debit": true, "credit": true}

type OrdersService struct {
	store   *store.Store
	printer *printer.Printer
}

func NewOrdersService(s *store.Store, p *printer.Printer) *OrdersService {
	return &OrdersService{store: s, printer: p}
}

// ItemInput is one line of a CreateOrderInput — an optional field is a
// pointer/zero-value so the same "was it provided" checks the JS
// validation ran (`item.productId`, Number(item.quantity)) translate
// directly.
type ItemInput struct {
	ProductID int64
	Quantity  int64
	Note      string
}

type CreateOrderInput struct {
	Source    string
	Reference string
	Items     []ItemInput
	Note      string
	Payment   *PaymentInput
}

type PaymentInput struct {
	HasPaymentMethod  bool
	PaymentMethod     string
	HasAmountReceived bool
	AmountReceived    float64
}

type validatedItem struct {
	productID int64
	name      string
	unitPrice float64
	quantity  int64
	note      *string
}

func (s *OrdersService) validateItems(items []ItemInput) ([]validatedItem, error) {
	if len(items) == 0 {
		return nil, NewValidationError("O pedido precisa ter pelo menos um item.")
	}

	result := make([]validatedItem, 0, len(items))
	for _, item := range items {
		if item.ProductID == 0 || item.Quantity <= 0 {
			return nil, NewValidationError("Item de pedido inválido.")
		}
		product, err := s.store.FindActiveProduct(item.ProductID)
		if err != nil {
			return nil, err
		}
		if product == nil {
			return nil, NewValidationError(fmt.Sprintf("Produto %d não encontrado ou inativo.", item.ProductID))
		}

		var note *string
		trimmed := strings.TrimSpace(item.Note)
		if trimmed != "" {
			note = &trimmed
		}

		result = append(result, validatedItem{
			productID: product.ID,
			name:      product.Name,
			unitPrice: product.Price,
			quantity:  item.Quantity,
			note:      note,
		})
	}
	return result, nil
}

func (s *OrdersService) validatePayment(payment *PaymentInput, total float64) (*validatedPayment, error) {
	if !payment.HasPaymentMethod || !validPaymentMethods[payment.PaymentMethod] {
		return nil, NewValidationError("Forma de pagamento inválida.")
	}

	register, err := s.store.OpenRegister()
	if err != nil {
		return nil, err
	}
	if register == nil {
		return nil, NewValidationError("Abra o caixa antes de registrar um pagamento.")
	}

	var amountReceived *float64
	changeDue := 0.0
	if payment.PaymentMethod == "cash" {
		if !payment.HasAmountReceived || centsOf(payment.AmountReceived) < centsOf(total) {
			return nil, NewValidationError("Valor recebido insuficiente.")
		}
		amountReceived = &payment.AmountReceived
		changeDue = round2(payment.AmountReceived - total)
	}

	return &validatedPayment{
		method:         payment.PaymentMethod,
		amountReceived: amountReceived,
		changeDue:      changeDue,
		cashRegisterID: register.ID,
	}, nil
}

type validatedPayment struct {
	method         string
	amountReceived *float64
	changeDue      float64
	cashRegisterID int64
}

func (s *OrdersService) createOrder(input CreateOrderInput) (int64, error) {
	source := input.Source
	if !validSources[source] {
		source = "counter"
	}

	items, err := s.validateItems(input.Items)
	if err != nil {
		return 0, err
	}

	total := 0.0
	for _, item := range items {
		total += item.unitPrice * float64(item.quantity)
	}

	var payment *validatedPayment
	if input.Payment != nil {
		payment, err = s.validatePayment(input.Payment, total)
		if err != nil {
			return 0, err
		}
	}

	date := clock.TodayLocal()
	createdAt := clock.NowLocal()

	tx, err := s.store.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	dailyNumber, err := s.store.NextDailyNumber(tx, date)
	if err != nil {
		return 0, err
	}

	var reference *string
	trimmedRef := strings.TrimSpace(input.Reference)
	if trimmedRef != "" {
		reference = &trimmedRef
	}
	var note *string
	trimmedNote := strings.TrimSpace(input.Note)
	if trimmedNote != "" {
		note = &trimmedNote
	}

	status := "open"
	var paymentMethod *string
	var amountReceived *float64
	var changeDue *float64
	var cashRegisterID *int64
	var paidAt *string
	if payment != nil {
		status = "paid"
		paymentMethod = &payment.method
		amountReceived = payment.amountReceived
		changeDue = &payment.changeDue
		cashRegisterID = &payment.cashRegisterID
		paidAt = &createdAt
	}

	orderID, err := s.store.InsertOrder(tx, store.NewOrder{
		DailyNumber:    dailyNumber,
		Date:           date,
		Source:         source,
		Reference:      reference,
		Status:         status,
		Total:          total,
		PaymentMethod:  paymentMethod,
		AmountReceived: amountReceived,
		ChangeDue:      changeDue,
		CashRegisterID: cashRegisterID,
		Note:           note,
		CreatedAt:      createdAt,
		PaidAt:         paidAt,
	})
	if err != nil {
		return 0, err
	}

	for _, item := range items {
		if err := s.store.InsertOrderItem(tx, orderID, store.NewOrderItem{
			ProductID: item.productID,
			Name:      item.name,
			UnitPrice: item.unitPrice,
			Quantity:  item.quantity,
			Note:      item.note,
		}); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return orderID, nil
}

type CreateOrderResult struct {
	Order    *store.Order           `json:"order"`
	Printing printer.NewOrderResult `json:"printing"`
	Receipt  *printer.Result        `json:"receipt"`
}

func (s *OrdersService) CreateOrderWithPrinting(input CreateOrderInput) (*CreateOrderResult, error) {
	orderID, err := s.createOrder(input)
	if err != nil {
		return nil, err
	}

	order, err := s.store.LoadOrder(orderID)
	if err != nil {
		return nil, err
	}

	printing := s.printer.PrintNewOrder(order)
	if err := s.store.MarkPrinted(orderID, printing.Success, clock.NowLocal()); err != nil {
		return nil, err
	}

	updated, err := s.store.LoadOrder(orderID)
	if err != nil {
		return nil, err
	}

	return &CreateOrderResult{Order: updated, Printing: printing, Receipt: printing.Receipt}, nil
}

type RegisterPaymentResult struct {
	Order   *store.Order   `json:"order"`
	Receipt printer.Result `json:"receipt"`
}

func (s *OrdersService) RegisterPayment(orderID int64, payment *PaymentInput) (*RegisterPaymentResult, error) {
	order, err := s.store.LoadOrder(orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, NewValidationError("Pedido não encontrado.")
	}
	if order.Status != "open" {
		return nil, NewValidationError("Este pedido não está aguardando pagamento.")
	}

	validated, err := s.validatePayment(payment, order.Total)
	if err != nil {
		return nil, err
	}
	paidAt := clock.NowLocal()

	if err := s.store.MarkPaid(orderID, validated.method, validated.amountReceived, validated.changeDue, validated.cashRegisterID, paidAt); err != nil {
		return nil, err
	}

	updated, err := s.store.LoadOrder(orderID)
	if err != nil {
		return nil, err
	}

	receipt := s.printer.PrintDocument("receipt", updated)
	return &RegisterPaymentResult{Order: updated, Receipt: receipt}, nil
}

func (s *OrdersService) CancelOrder(orderID int64, reason string) (*store.Order, error) {
	order, err := s.store.LoadOrder(orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, NewValidationError("Pedido não encontrado.")
	}
	if order.Status == "canceled" {
		return nil, NewValidationError("Este pedido já está cancelado.")
	}
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return nil, NewValidationError("Informe o motivo do cancelamento.")
	}

	if err := s.store.MarkCanceled(orderID, trimmed, clock.NowLocal()); err != nil {
		return nil, err
	}
	return s.store.LoadOrder(orderID)
}

func (s *OrdersService) Reprint(orderID int64, docType string) (*printer.Result, error) {
	order, err := s.store.LoadOrder(orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, NewValidationError("Pedido não encontrado.")
	}

	result := s.printer.PrintDocument(docType, order)
	if result.Success {
		if err := s.store.MarkPrinted(orderID, true, clock.NowLocal()); err != nil {
			return nil, err
		}
	}
	return &result, nil
}

func (s *OrdersService) LoadOrder(orderID int64) (*store.Order, error) {
	return s.store.LoadOrder(orderID)
}

func (s *OrdersService) ListForDay(date, status string) ([]store.Order, error) {
	if date == "" {
		date = clock.TodayLocal()
	}
	return s.store.ListOrdersForDay(date, status)
}
