// Package store defines the JSON-facing models and the SQL access layer.
// Struct field JSON tags mirror the camelCase keys the Node backend
// produced via toCamelCase(), and nullable DB columns are pointers so they
// serialize to JSON null exactly as the JS objects did.
package store

type Category struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortOrder int64  `json:"sortOrder"`
	Active    int64  `json:"active"`
}

type Product struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Price          float64 `json:"price"`
	CategoryID     *int64  `json:"categoryId"`
	Active         int64   `json:"active"`
	AvailableToday int64   `json:"availableToday"`
	SortOrder      int64   `json:"sortOrder"`
}

type CashRegister struct {
	ID             int64    `json:"id"`
	OpeningAmount  float64  `json:"openingAmount"`
	OpenedAt       string   `json:"openedAt"`
	ClosedAt       *string  `json:"closedAt"`
	CountedAmount  *float64 `json:"countedAmount"`
	TotalCash      *float64 `json:"totalCash"`
	TotalPix       *float64 `json:"totalPix"`
	TotalDebit     *float64 `json:"totalDebit"`
	TotalCredit    *float64 `json:"totalCredit"`
	TotalCashOut   *float64 `json:"totalCashOut"`
	TotalCashIn    *float64 `json:"totalCashIn"`
	ExpectedAmount *float64 `json:"expectedAmount"`
	Difference     *float64 `json:"difference"`
	Status         string   `json:"status"`
}

type CashRegisterMovement struct {
	ID             int64   `json:"id"`
	CashRegisterID int64   `json:"cashRegisterId"`
	Type           string  `json:"type"`
	Amount         float64 `json:"amount"`
	Reason         *string `json:"reason"`
	CreatedAt      string  `json:"createdAt"`
}

type Order struct {
	ID                 int64       `json:"id"`
	DailyNumber        int64       `json:"dailyNumber"`
	Date               string      `json:"date"`
	Source             string      `json:"source"`
	Reference          *string     `json:"reference"`
	Status             string      `json:"status"`
	Total              float64     `json:"total"`
	PaymentMethod      *string     `json:"paymentMethod"`
	AmountReceived     *float64    `json:"amountReceived"`
	ChangeDue          *float64    `json:"changeDue"`
	CashRegisterID     *int64      `json:"cashRegisterId"`
	Note               *string     `json:"note"`
	CreatedAt          string      `json:"createdAt"`
	PaidAt             *string     `json:"paidAt"`
	CanceledAt         *string     `json:"canceledAt"`
	CancellationReason *string     `json:"cancellationReason"`
	PrintedAt          *string     `json:"printedAt"`
	PrintAttempts      int64       `json:"printAttempts"`
	Items              []OrderItem `json:"items"`
}

type OrderItem struct {
	ID        int64   `json:"id"`
	OrderID   int64   `json:"orderId"`
	ProductID *int64  `json:"productId"`
	Name      string  `json:"name"`
	UnitPrice float64 `json:"unitPrice"`
	Quantity  int64   `json:"quantity"`
	Note      *string `json:"note"`
}
