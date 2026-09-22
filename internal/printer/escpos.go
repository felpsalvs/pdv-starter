// Package printer sends kitchen tickets, packaging labels and receipts to a
// thermal printer, replacing node-thermal-printer + the OS raw-print
// shellouts from src/printer.js with an equivalent pure-Go implementation:
// a small ESC/POS byte encoder here, and the same "write temp file, hand it
// to the OS printing pipe" approach in send.go.
package printer

import (
	"bytes"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// builder accumulates ESC/POS commands for one ticket. Text is encoded as
// CP860 (Portuguese), matching node-thermal-printer's
// CharacterSet.PC860_PORTUGUESE, so accented characters print correctly.
type builder struct {
	buf     bytes.Buffer
	encoder *encoding.Encoder
}

func newBuilder() *builder {
	b := &builder{encoder: encoding.ReplaceUnsupported(charmap.CodePage860.NewEncoder())}
	b.buf.Write([]byte{0x1B, 0x40})       // ESC @ : initialize
	b.buf.Write([]byte{0x1B, 0x74, 0x03}) // ESC t 3 : PC860 (Portuguese) code table
	return b
}

func (b *builder) text(s string) {
	encoded, err := b.encoder.String(s)
	if err != nil {
		// ReplaceUnsupported guarantees no error, but fall back to the raw
		// string rather than dropping the line if that ever changes.
		encoded = s
	}
	b.buf.WriteString(encoded)
}

func (b *builder) Println(s string) {
	b.text(s)
	b.buf.WriteByte('\n')
}

func (b *builder) NewLine() {
	b.buf.WriteByte('\n')
}

func (b *builder) AlignLeft() {
	b.buf.Write([]byte{0x1B, 0x61, 0x00})
}

func (b *builder) AlignCenter() {
	b.buf.Write([]byte{0x1B, 0x61, 0x01})
}

func (b *builder) Bold(on bool) {
	if on {
		b.buf.Write([]byte{0x1B, 0x45, 0x01})
	} else {
		b.buf.Write([]byte{0x1B, 0x45, 0x00})
	}
}

// SetTextSize doubles both width and height (the only size the tickets use,
// matching the original printer.setTextSize(1, 1) calls).
func (b *builder) SetTextSize() {
	b.buf.Write([]byte{0x1D, 0x21, 0x11})
}

func (b *builder) SetTextNormal() {
	b.buf.Write([]byte{0x1D, 0x21, 0x00})
}

func (b *builder) DrawLine() {
	b.Println("--------------------------------")
}

func (b *builder) Cut() {
	b.buf.Write([]byte{0x1D, 0x56, 0x00})
}

func (b *builder) Bytes() []byte {
	return b.buf.Bytes()
}
