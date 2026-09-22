package service

import "testing"

func TestOpenRegisterTwiceFails(t *testing.T) {
	_, _, cashRegister := newTestServices(t)
	if _, err := cashRegister.Open(100, true); err != nil {
		t.Fatal(err)
	}
	_, err := cashRegister.Open(50, true)
	assertValidationError(t, err, "A cash register is already open.")
}

func TestOpenRegisterRejectsNegativeAmount(t *testing.T) {
	_, _, cashRegister := newTestServices(t)
	_, err := cashRegister.Open(-1, true)
	assertValidationError(t, err, "Enter a valid opening amount.")
}

func TestMovementRequiresOpenRegister(t *testing.T) {
	_, _, cashRegister := newTestServices(t)
	_, err := cashRegister.RegisterMovement("cash_out", 10, true, "gelo")
	assertValidationError(t, err, "There is no open cash register.")
}

func TestMovementValidation(t *testing.T) {
	_, _, cashRegister := newTestServices(t)
	if _, err := cashRegister.Open(100, true); err != nil {
		t.Fatal(err)
	}

	if _, err := cashRegister.RegisterMovement("invalid", 10, true, "x"); err == nil {
		t.Error("expected error for invalid movement type")
	}
	if _, err := cashRegister.RegisterMovement("cash_out", 0, true, "x"); err == nil {
		t.Error("expected error for zero amount")
	}
	if _, err := cashRegister.RegisterMovement("cash_out", 10, true, ""); err == nil {
		t.Error("expected error for empty reason")
	}
}

func TestCloseRegisterComputesExpectedAndDifference(t *testing.T) {
	st, orders, cashRegister := newTestServices(t)
	productID := seedProduct(t, st, "Caldo verde", 18.5)

	if _, err := cashRegister.Open(100, true); err != nil {
		t.Fatal(err)
	}
	if _, err := orders.CreateOrderWithPrinting(CreateOrderInput{
		Source: "counter",
		Items:  []ItemInput{{ProductID: productID, Quantity: 2}}, // 37.00
		Payment: &PaymentInput{
			HasPaymentMethod: true, PaymentMethod: "cash",
			HasAmountReceived: true, AmountReceived: 37,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := cashRegister.RegisterMovement("cash_out", 10, true, "gelo"); err != nil {
		t.Fatal(err)
	}

	// expected = opening(100) + cashSales(37) + cashIn(0) - cashOut(10) = 127
	result, err := cashRegister.Close(125, true)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if result.Register.ExpectedAmount == nil || *result.Register.ExpectedAmount != 127 {
		t.Errorf("expectedAmount = %v, want 127", result.Register.ExpectedAmount)
	}
	if result.Difference != -2 {
		t.Errorf("difference = %v, want -2", result.Difference)
	}
	if result.Register.Status != "closed" {
		t.Errorf("status = %q, want closed", result.Register.Status)
	}

	closures, err := cashRegister.ListClosures()
	if err != nil {
		t.Fatal(err)
	}
	if len(closures) != 1 {
		t.Fatalf("closures = %d, want 1", len(closures))
	}
}

func TestCloseRegisterWithoutOpenOneFails(t *testing.T) {
	_, _, cashRegister := newTestServices(t)
	_, err := cashRegister.Close(0, true)
	assertValidationError(t, err, "There is no open cash register to close.")
}

func TestRound2MatchesJSMathRound(t *testing.T) {
	cases := map[float64]float64{
		10.126:          10.13, // ordinary rounding up
		-1.004:          -1.0,  // ordinary rounding down towards zero
		0.125:           0.13,  // exact tie rounds up (towards +Infinity)
		-0.125:          -0.12, // exact tie still rounds towards +Infinity, i.e. less negative
		0.1 + 0.2 - 0.3: 0,     // floating-point noise around zero should still land on 0
	}
	for input, want := range cases {
		if got := round2(input); got != want {
			t.Errorf("round2(%v) = %v, want %v", input, got, want)
		}
	}
}
