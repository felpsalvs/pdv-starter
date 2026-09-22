package printer

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// formatBRL mirrors value.toLocaleString('pt-BR', { style: 'currency',
// currency: 'BRL' }): "R$ 1.234,56" — dot as the thousands separator,
// comma as the decimal separator.
func formatBRL(value float64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}

	cents := int64(value*100 + 0.5)
	whole := cents / 100
	frac := cents % 100

	wholeStr := strconv.FormatInt(whole, 10)
	grouped := groupThousands(wholeStr)

	return fmt.Sprintf("%sR$ %s,%02d", sign, grouped, frac)
}

func groupThousands(digits string) string {
	n := len(digits)
	if n <= 3 {
		return digits
	}
	var parts []string
	for n > 3 {
		parts = append([]string{digits[n-3 : n]}, parts...)
		n -= 3
	}
	parts = append([]string{digits[:n]}, parts...)
	return strings.Join(parts, ".")
}

// formatDateTimePtBR mirrors new Date(dbTimestamp.replace(' ', 'T'))
// .toLocaleString('pt-BR'): "dd/mm/aaaa, HH:MM:SS". dbTimestamp is a local
// wall-clock string ("YYYY-MM-DD HH:MM:SS", the same shape clock.NowLocal()
// produces), so it's parsed and re-formatted without any timezone
// conversion.
func formatDateTimePtBR(dbTimestamp string) string {
	t, err := time.Parse("2006-01-02 15:04:05", dbTimestamp)
	if err != nil {
		return dbTimestamp
	}
	return t.Format("02/01/2006, 15:04:05")
}
