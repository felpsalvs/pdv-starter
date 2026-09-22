package printer

import (
	"encoding/json"
	"os"

	"github.com/felpsalvs/pdv-starter/internal/store"
)

const notConfiguredError = "Impressora não configurada. Copie printer.config.example.json para printer.config.json e informe o nome da impressora."

type Printer struct {
	ConfigPath string
}

func New(configPath string) *Printer {
	return &Printer{ConfigPath: configPath}
}

type printerConfig struct {
	PrinterName string `json:"printerName"`
}

func (p *Printer) readConfig() *printerConfig {
	raw, err := os.ReadFile(p.ConfigPath)
	if err != nil {
		return nil
	}
	var cfg printerConfig
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.PrinterName == "" {
		return nil
	}
	return &cfg
}

// Result is one print job's outcome, matching the Node backend's
// { success, reason } shape.
type Result struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}

// NewOrderResult is what POST /api/orders returns for the `printing` field.
type NewOrderResult struct {
	Success bool    `json:"success"`
	Kitchen Result  `json:"kitchen"`
	Label   Result  `json:"label"`
	Receipt *Result `json:"receipt"`
}

type job struct {
	docType string
	order   *store.Order
}

func (p *Printer) printBatch(jobs []job) []Result {
	cfg := p.readConfig()
	if cfg == nil {
		results := make([]Result, len(jobs))
		for i := range results {
			results[i] = Result{Success: false, Reason: notConfiguredError}
		}
		return results
	}

	results := make([]Result, len(jobs))
	for i, j := range jobs {
		results[i] = p.printOne(cfg.PrinterName, j)
	}
	return results
}

func (p *Printer) printOne(printerName string, j job) Result {
	b := newBuilder()
	switch j.docType {
	case "kitchen":
		buildKitchenTicket(b, j.order)
	case "label":
		for _, item := range j.order.Items {
			for unit := int64(0); unit < item.Quantity; unit++ {
				buildLabel(b, j.order, item)
			}
		}
	case "receipt":
		buildReceipt(b, j.order)
	default:
		return Result{Success: false, Reason: "Unknown document type: " + j.docType}
	}

	if err := printRawBytes(printerName, b.Bytes()); err != nil {
		return Result{Success: false, Reason: "Falha ao imprimir: " + err.Error()}
	}
	return Result{Success: true}
}

// PrintDocument prints a single document of the given type (kitchen, label
// or receipt) for an order — used for the pay/reprint endpoints.
func (p *Printer) PrintDocument(docType string, order *store.Order) Result {
	results := p.printBatch([]job{{docType: docType, order: order}})
	return results[0]
}

// PrintNewOrder prints the kitchen ticket and packaging labels for a new
// order, plus the receipt when it was paid at creation time — mirroring
// printNewOrder in src/printer.js.
func (p *Printer) PrintNewOrder(order *store.Order) NewOrderResult {
	jobs := []job{
		{docType: "kitchen", order: order},
		{docType: "label", order: order},
	}
	printReceipt := order.Status == "paid"
	if printReceipt {
		jobs = append(jobs, job{docType: "receipt", order: order})
	}

	results := p.printBatch(jobs)
	kitchen, label := results[0], results[1]

	var receipt *Result
	receiptOK := true
	if printReceipt {
		receipt = &results[2]
		receiptOK = receipt.Success
	}

	return NewOrderResult{
		Success: kitchen.Success && label.Success && receiptOK,
		Kitchen: kitchen,
		Label:   label,
		Receipt: receipt,
	}
}
