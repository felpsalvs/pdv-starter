package printer

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/felpsalvs/pdv-starter/internal/store"
)

func TestPrintBatchWithoutConfigReportsNotConfigured(t *testing.T) {
	p := New(filepath.Join(t.TempDir(), "printer.config.json"))
	order := &store.Order{DailyNumber: 1, Source: "counter", Status: "open", CreatedAt: "2024-01-05 12:00:00"}

	result := p.PrintNewOrder(order)
	if result.Success {
		t.Fatal("expected failure without a printer config")
	}
	if result.Kitchen.Success || result.Kitchen.Reason == "" {
		t.Errorf("kitchen result = %+v, want a failure with a reason", result.Kitchen)
	}
}

func TestKitchenTicketContainsOrderContents(t *testing.T) {
	note := "sem cebola"
	order := &store.Order{
		DailyNumber: 7,
		Source:      "table",
		Reference:   strPtr("5"),
		Status:      "open",
		CreatedAt:   "2024-01-05 12:30:00",
		Items: []store.OrderItem{
			{Name: "Caldo verde", Quantity: 2, Note: &note},
		},
	}

	b := newBuilder()
	buildKitchenTicket(b, order)
	out := b.Bytes()

	for _, want := range []string{"SENHA 7", "Mesa 5", "2x Caldo verde", "Obs: sem cebola"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("kitchen ticket missing %q\n%s", want, out)
		}
	}
}

func TestReceiptFormatsCurrencyAndChange(t *testing.T) {
	amountReceived := 50.0
	changeDue := 13.0
	order := &store.Order{
		DailyNumber:    3,
		Source:         "counter",
		Status:         "paid",
		Total:          37,
		PaymentMethod:  strPtr("cash"),
		AmountReceived: &amountReceived,
		ChangeDue:      &changeDue,
		Items: []store.OrderItem{
			{Name: "Caldo verde", UnitPrice: 18.5, Quantity: 2},
		},
	}

	b := newBuilder()
	buildReceipt(b, order)
	out := b.Bytes()

	for _, want := range []string{"Total: R$ 37,00", "Recebido: R$ 50,00", "Troco: R$ 13,00", "Dinheiro"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("receipt missing %q\n%s", want, out)
		}
	}
}

func TestFormatBRL(t *testing.T) {
	cases := map[float64]string{
		0:       "R$ 0,00",
		18.5:    "R$ 18,50",
		1234.56: "R$ 1.234,56",
		-5:      "-R$ 5,00",
	}
	for input, want := range cases {
		if got := formatBRL(input); got != want {
			t.Errorf("formatBRL(%v) = %q, want %q", input, got, want)
		}
	}
}

func strPtr(s string) *string { return &s }
