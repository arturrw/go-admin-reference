package httpapi

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"strconv"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

//go:embed invoice.html
var invoiceHTML string

var invoiceTmpl = template.Must(template.New("invoice").Funcs(template.FuncMap{
	"usd":  d.USD,
	"date": func(t time.Time) string { return t.UTC().Format("2 Jan 2006") },
	"time": func(t time.Time) string { return t.UTC().Format("2 Jan 2006, 15:04 UTC") },
	"line": func(it d.OrderItem) string { return d.USD(it.PriceCents * int64(it.Qty)) },
}).Parse(invoiceHTML))

type invoiceView struct {
	Seller   string
	SellerAt string
	Number   string
	Order    d.Order
	Customer d.Customer
	Units    int
	Now      time.Time
	Print    bool
}

// invoice: GET /api/v1/orders/{id}/invoice is a printable invoice page. The
// UI opens it in a new tab; "Print / Save as PDF" turns it into a PDF with the
// browser's own engine, so there is no PDF library in the binary.
func (s *server) invoice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	o, err := s.store.GetOrder(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	det, err := s.store.GetCustomer(r.Context(), o.Customer.ID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	o.FillTotals()
	set := s.settings(r.Context())
	v := invoiceView{
		Seller: set.ServiceName, SellerAt: set.PublicBaseURL, Number: "INV-" + strconv.FormatInt(o.ID, 10),
		Order: o, Customer: det.Customer, Now: time.Now(), Print: r.URL.Query().Get("print") == "1",
	}
	for _, it := range o.Items {
		v.Units += it.Qty
	}
	var buf bytes.Buffer
	if err := invoiceTmpl.Execute(&buf, v); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(buf.Bytes())
}
