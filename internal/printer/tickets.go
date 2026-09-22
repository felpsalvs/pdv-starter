package printer

import (
	"fmt"

	"github.com/felpsalvs/pdv-starter/internal/store"
)

var paymentMethodLabels = map[string]string{
	"cash":   "Dinheiro",
	"pix":    "Pix",
	"debit":  "Cartão débito",
	"credit": "Cartão crédito",
}

func referenceLine(order *store.Order) string {
	if order.Source == "table" {
		ref := "?"
		if order.Reference != nil && *order.Reference != "" {
			ref = *order.Reference
		}
		return "Mesa " + ref
	}
	if order.Reference != nil && *order.Reference != "" {
		return *order.Reference
	}
	return "Balcão"
}

func buildKitchenTicket(b *builder, order *store.Order) {
	b.AlignCenter()
	b.SetTextSize()
	b.Bold(true)
	b.Println(fmt.Sprintf("SENHA %d", order.DailyNumber))
	b.SetTextNormal()
	b.Println("PEDIDO PARA A COZINHA")
	b.Bold(false)
	b.DrawLine()

	b.AlignLeft()
	b.Println(fmt.Sprintf("%s · %s", referenceLine(order), formatDateTimePtBR(order.CreatedAt)))
	b.NewLine()

	for _, item := range order.Items {
		b.Bold(true)
		b.Println(fmt.Sprintf("%dx %s", item.Quantity, item.Name))
		b.Bold(false)
		if item.Note != nil && *item.Note != "" {
			b.Println("   Obs: " + *item.Note)
		}
	}

	if order.Note != nil && *order.Note != "" {
		b.NewLine()
		b.Println("Obs geral: " + *order.Note)
	}

	b.DrawLine()
	b.Cut()
}

func buildLabel(b *builder, order *store.Order, item store.OrderItem) {
	b.AlignCenter()
	b.Bold(true)
	b.SetTextSize()
	b.Println(fmt.Sprintf("%d", order.DailyNumber))
	b.SetTextNormal()
	b.Println(referenceLine(order))
	b.Bold(false)
	b.DrawLine()
	b.Bold(true)
	b.Println(item.Name)
	b.Bold(false)
	if item.Note != nil && *item.Note != "" {
		b.Println(*item.Note)
	}
	b.Cut()
}

func buildReceipt(b *builder, order *store.Order) {
	b.AlignCenter()
	b.Bold(true)
	b.Println("RECIBO")
	b.Bold(false)
	b.DrawLine()

	b.AlignLeft()
	b.Println(fmt.Sprintf("Pedido #%d · %s", order.DailyNumber, referenceLine(order)))
	b.NewLine()

	for _, item := range order.Items {
		b.Println(fmt.Sprintf("%dx %s", item.Quantity, item.Name))
		b.Println("   " + formatBRL(item.UnitPrice*float64(item.Quantity)))
	}

	b.DrawLine()
	b.Bold(true)
	b.Println("Total: " + formatBRL(order.Total))
	b.Bold(false)
	method := ""
	if order.PaymentMethod != nil {
		method = *order.PaymentMethod
	}
	label, ok := paymentMethodLabels[method]
	if !ok {
		label = method
	}
	b.Println("Pagamento: " + label)
	if method == "cash" && order.AmountReceived != nil {
		b.Println("Recebido: " + formatBRL(*order.AmountReceived))
		changeDue := 0.0
		if order.ChangeDue != nil {
			changeDue = *order.ChangeDue
		}
		b.Println("Troco: " + formatBRL(changeDue))
	}

	b.DrawLine()
	b.Cut()
}
