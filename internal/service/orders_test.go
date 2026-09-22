package service

import "testing"

func TestCreateOrderRequiresItems(t *testing.T) {
	_, orders, _ := newTestServices(t)

	_, err := orders.CreateOrderWithPrinting(CreateOrderInput{Source: "counter"})
	assertValidationError(t, err, "O pedido precisa ter pelo menos um item.")
}

func TestCreateOrderRejectsInactiveOrMissingProduct(t *testing.T) {
	_, orders, _ := newTestServices(t)

	_, err := orders.CreateOrderWithPrinting(CreateOrderInput{
		Source: "counter",
		Items:  []ItemInput{{ProductID: 999, Quantity: 1}},
	})
	assertValidationError(t, err, "Produto 999 não encontrado ou inativo.")
}

func TestCreateOrderPaymentRequiresOpenRegister(t *testing.T) {
	st, orders, _ := newTestServices(t)
	productID := seedProduct(t, st, "Caldo verde", 18.5)

	_, err := orders.CreateOrderWithPrinting(CreateOrderInput{
		Source: "counter",
		Items:  []ItemInput{{ProductID: productID, Quantity: 1}},
		Payment: &PaymentInput{
			HasPaymentMethod: true, PaymentMethod: "cash",
			HasAmountReceived: true, AmountReceived: 20,
		},
	})
	assertValidationError(t, err, "Abra o caixa antes de registrar um pagamento.")
}

func TestCreateOrderCashPaymentComputesChange(t *testing.T) {
	st, orders, cashRegister := newTestServices(t)
	productID := seedProduct(t, st, "Caldo verde", 18.5)

	if _, err := cashRegister.Open(100, true); err != nil {
		t.Fatalf("Open register: %v", err)
	}

	result, err := orders.CreateOrderWithPrinting(CreateOrderInput{
		Source: "counter",
		Items:  []ItemInput{{ProductID: productID, Quantity: 2}}, // 37.00
		Payment: &PaymentInput{
			HasPaymentMethod: true, PaymentMethod: "cash",
			HasAmountReceived: true, AmountReceived: 50,
		},
	})
	if err != nil {
		t.Fatalf("CreateOrderWithPrinting: %v", err)
	}
	if result.Order.Status != "paid" {
		t.Errorf("status = %q, want paid", result.Order.Status)
	}
	if result.Order.Total != 37 {
		t.Errorf("total = %v, want 37", result.Order.Total)
	}
	if result.Order.ChangeDue == nil || *result.Order.ChangeDue != 13 {
		t.Errorf("changeDue = %v, want 13", result.Order.ChangeDue)
	}
	// The printer isn't configured in this test, so every job fails and
	// print_attempts must still be recorded — mirrors markPrinted(false).
	if result.Order.PrintAttempts != 1 {
		t.Errorf("printAttempts = %d, want 1", result.Order.PrintAttempts)
	}
	if result.Printing.Success {
		t.Error("printing.success = true, want false (no printer configured)")
	}
}

func TestCreateOrderInsufficientCashRejected(t *testing.T) {
	st, orders, cashRegister := newTestServices(t)
	productID := seedProduct(t, st, "Caldo verde", 18.5)
	if _, err := cashRegister.Open(0, true); err != nil {
		t.Fatal(err)
	}

	_, err := orders.CreateOrderWithPrinting(CreateOrderInput{
		Source: "counter",
		Items:  []ItemInput{{ProductID: productID, Quantity: 1}},
		Payment: &PaymentInput{
			HasPaymentMethod: true, PaymentMethod: "cash",
			HasAmountReceived: true, AmountReceived: 10,
		},
	})
	assertValidationError(t, err, "Valor recebido insuficiente.")
}

func TestDailyNumberIncrementsPerDate(t *testing.T) {
	st, orders, _ := newTestServices(t)
	productID := seedProduct(t, st, "Caldo verde", 18.5)

	for i, want := range []int64{1, 2, 3} {
		result, err := orders.CreateOrderWithPrinting(CreateOrderInput{
			Source: "counter",
			Items:  []ItemInput{{ProductID: productID, Quantity: 1}},
		})
		if err != nil {
			t.Fatalf("order %d: %v", i, err)
		}
		if result.Order.DailyNumber != want {
			t.Errorf("order %d: dailyNumber = %d, want %d", i, result.Order.DailyNumber, want)
		}
	}
}

func TestCancelOrderTwiceFails(t *testing.T) {
	st, orders, _ := newTestServices(t)
	productID := seedProduct(t, st, "Caldo verde", 18.5)
	result, err := orders.CreateOrderWithPrinting(CreateOrderInput{
		Source: "table", Reference: "5",
		Items: []ItemInput{{ProductID: productID, Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := orders.CancelOrder(result.Order.ID, "cliente desistiu"); err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	_, err = orders.CancelOrder(result.Order.ID, "de novo")
	assertValidationError(t, err, "Este pedido já está cancelado.")
}

func TestCancelOrderRequiresReason(t *testing.T) {
	st, orders, _ := newTestServices(t)
	productID := seedProduct(t, st, "Caldo verde", 18.5)
	result, err := orders.CreateOrderWithPrinting(CreateOrderInput{
		Source: "table", Reference: "5",
		Items: []ItemInput{{ProductID: productID, Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = orders.CancelOrder(result.Order.ID, "  ")
	assertValidationError(t, err, "Informe o motivo do cancelamento.")
}

func TestRegisterPaymentOnAlreadyPaidOrderFails(t *testing.T) {
	st, orders, cashRegister := newTestServices(t)
	productID := seedProduct(t, st, "Caldo verde", 18.5)
	if _, err := cashRegister.Open(0, true); err != nil {
		t.Fatal(err)
	}
	result, err := orders.CreateOrderWithPrinting(CreateOrderInput{
		Source: "counter",
		Items:  []ItemInput{{ProductID: productID, Quantity: 1}},
		Payment: &PaymentInput{
			HasPaymentMethod: true, PaymentMethod: "pix",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = orders.RegisterPayment(result.Order.ID, &PaymentInput{HasPaymentMethod: true, PaymentMethod: "pix"})
	assertValidationError(t, err, "Este pedido não está aguardando pagamento.")
}

func TestReprintMissingOrderFails(t *testing.T) {
	_, orders, _ := newTestServices(t)
	_, err := orders.Reprint(999, "receipt")
	assertValidationError(t, err, "Pedido não encontrado.")
}

func assertValidationError(t *testing.T, err error, wantMessage string) {
	t.Helper()
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("err = %v (%T), want *ValidationError", err, err)
	}
	if ve.Message != wantMessage {
		t.Errorf("message = %q, want %q", ve.Message, wantMessage)
	}
}
