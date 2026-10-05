package httpapi

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// CSV export and import. Money columns are decimal dollars ("124.99"), lists
// are separated by "; ". Exports start with a UTF-8 BOM so Excel detects the
// encoding; the product export doubles as the import template.

const (
	maxImportBytes = 2 << 20
	maxImportRows  = 1000
	// Exports are capped well above the seed size so one request cannot load
	// an unbounded table.
	maxExportRows = 50_000

	utf8BOM = "\xEF\xBB\xBF"
)

var productColumns = []string{
	"id", "name", "sku", "category", "vendor", "tags", "price", "compare_at_price", "cost",
	"stock", "weight_grams", "status", "sold_30d", "rating", "description", "image_url",
}

func writeCSV(w http.ResponseWriter, name string, header []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.csv"`, name, time.Now().Format("2006-01-02")))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, utf8BOM)
	cw := csv.NewWriter(w)
	_ = cw.Write(header)
	_ = cw.WriteAll(rows)
}

func dollars(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// parseDollars accepts "124.99", "$1,249.5" or "" (= 0) and returns cents.
func parseDollars(s string) (int64, error) {
	s = strings.NewReplacer("$", "", ",", "", " ", "").Replace(s)
	if s == "" {
		return 0, nil
	}
	whole, frac, _ := strings.Cut(s, ".")
	if len(frac) > 2 {
		return 0, errors.New("has more than 2 decimals")
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || strings.HasPrefix(whole, "-") {
		return 0, errors.New("is not a valid amount")
	}
	f := int64(0)
	if frac != "" {
		if f, err = strconv.ParseInt(frac+strings.Repeat("0", 2-len(frac)), 10, 64); err != nil {
			return 0, errors.New("is not a valid amount")
		}
	}
	return w*100 + f, nil
}

func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ';' || r == '|' })
}

func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// ── Exports ─────────────────────────────────────────────────────────────────

func (s *server) exportProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListProducts(r.Context(), d.ProductFilter{
		Query: q.Get("q"), Category: d.Category(q.Get("category")), Status: d.ProductStatus(q.Get("status")), Sort: q.Get("sort"),
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	rows := make([][]string, 0, len(items))
	for _, p := range items {
		compareAt := ""
		if p.CompareAtCents > 0 {
			compareAt = dollars(p.CompareAtCents)
		}
		img := p.ImageURL()
		if strings.HasPrefix(img, "/") {
			img = baseURL(r) + img
		}
		rows = append(rows, []string{
			strconv.FormatInt(p.ID, 10), p.Name, p.SKU, string(p.Category), p.Vendor, strings.Join(p.Tags, "; "),
			dollars(p.PriceCents), compareAt, dollars(p.CostCents), strconv.Itoa(p.Stock), strconv.Itoa(p.WeightGrams),
			string(p.Status), strconv.Itoa(p.Sold30d), strconv.FormatFloat(p.Rating, 'f', 1, 64), p.Description, img,
		})
	}
	writeCSV(w, "products", productColumns, rows)
}

func (s *server) exportOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, _, err := s.store.ListOrders(r.Context(), d.OrderFilter{
		Query: q.Get("q"), Status: d.OrderStatus(q.Get("status")), CustomerID: int64(queryInt(r, "customer", 0)), Limit: maxExportRows,
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	rows := make([][]string, 0, len(items))
	for _, o := range items {
		units, lines := 0, make([]string, 0, len(o.Items))
		for _, it := range o.Items {
			units += it.Qty
			lines = append(lines, fmt.Sprintf("%d× %s (%s)", it.Qty, it.Name, it.SKU))
		}
		var refundReason, refundedBy, refundedAt string
		if o.Refund != nil {
			refundReason, refundedBy, refundedAt = o.Refund.Reason, o.Refund.By, stamp(o.Refund.At)
		}
		rows = append(rows, []string{
			strconv.FormatInt(o.ID, 10), stamp(o.PlacedAt), string(o.Status), strconv.FormatInt(o.Customer.ID, 10),
			o.Customer.Name, o.Customer.Email, o.Customer.Country, o.Payment, strconv.Itoa(units), strings.Join(lines, "; "),
			dollars(o.TotalCents), refundReason, refundedBy, refundedAt,
		})
	}
	writeCSV(w, "orders", []string{
		"id", "placed_at", "status", "customer_id", "customer_name", "customer_email", "country", "payment", "units", "items", "total",
		"refund_reason", "refunded_by", "refunded_at",
	}, rows)
}

func (s *server) exportCustomers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListCustomers(r.Context(), d.CustomerFilter{Query: q.Get("q"), Segment: q.Get("segment")})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	rows := make([][]string, 0, len(items))
	for _, c := range items {
		rows = append(rows, []string{
			strconv.FormatInt(c.ID, 10), c.Name, c.Email, c.Phone, c.Country, c.Address.City, c.Address.Line1, c.Address.PostalCode,
			c.Segment, strconv.Itoa(c.Orders), dollars(c.LTVCents), strings.Join(c.Tags, "; "),
			strconv.FormatBool(c.AcceptsMarketing), c.Source, stamp(c.CreatedAt), stamp(c.LastOrderAt), stamp(c.LastSeenAt),
		})
	}
	writeCSV(w, "customers", []string{
		"id", "name", "email", "phone", "country", "city", "address", "postal_code", "segment", "orders", "lifetime_value",
		"tags", "accepts_marketing", "source", "created_at", "last_order_at", "last_seen_at",
	}, rows)
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// ── Import ──────────────────────────────────────────────────────────────────

type importRowError struct {
	Row     int    `json:"row"` // 1-based line number in the file, header = 1
	SKU     string `json:"sku,omitempty"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type importResult struct {
	Created int              `json:"created"`
	Updated int              `json:"updated"`
	Errors  []importRowError `json:"errors"`
}

// importProducts upserts products by SKU from a CSV file (multipart "file"
// field or a text/csv body). Columns are matched by header name, in any
// order; unknown columns (id, sold_30d, image_url…) are ignored. For existing
// SKUs, columns missing from the file keep their current values.
//
// The file is validated as a whole first: if any row is invalid nothing is
// written and the response lists every problem.
func (s *server) importProducts(w http.ResponseWriter, r *http.Request) {
	data, ok := readImportBody(w, r)
	if !ok {
		return
	}
	cr := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte(utf8BOM))))
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not parse CSV: "+err.Error())
		return
	}
	if len(records) < 2 {
		writeError(w, http.StatusBadRequest, "the file needs a header row and at least one product")
		return
	}
	if len(records)-1 > maxImportRows {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("at most %d products per import", maxImportRows))
		return
	}

	col := map[string]int{}
	for i, h := range records[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	if _, ok := col["sku"]; !ok {
		writeError(w, http.StatusBadRequest, `the header must include a "sku" column (export products to get a template)`)
		return
	}

	existing, err := s.store.ListProducts(r.Context(), d.ProductFilter{})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	bySKU := make(map[string]d.Product, len(existing))
	for _, p := range existing {
		bySKU[strings.ToUpper(p.SKU)] = p
	}

	type planned struct {
		id int64 // 0 = create
		in d.ProductInput
	}
	var plan []planned
	var errs []importRowError
	seen := map[string]int{}

	for i, rec := range records[1:] {
		line := i + 2
		get := func(name string) (string, bool) {
			j, ok := col[name]
			if !ok || j >= len(rec) {
				return "", false
			}
			return strings.TrimSpace(rec[j]), true
		}
		if strings.TrimSpace(strings.Join(rec, "")) == "" {
			continue // blank line
		}
		sku, _ := get("sku")
		sku = strings.ToUpper(sku)
		fail := func(field, msg string) {
			errs = append(errs, importRowError{Row: line, SKU: sku, Field: field, Message: msg})
		}
		if sku == "" {
			fail("sku", "is required")
			continue
		}
		if prev, dup := seen[sku]; dup {
			fail("sku", fmt.Sprintf("duplicates row %d", prev))
			continue
		}
		seen[sku] = line

		cur, exists := bySKU[sku]
		in := d.ProductInput{Status: d.ProductDraft, Tags: []string{}}
		if exists {
			in = d.ProductInput{
				Name: cur.Name, SKU: cur.SKU, Category: cur.Category, Vendor: cur.Vendor, Tags: cur.Tags,
				PriceCents: cur.PriceCents, CompareAtCents: cur.CompareAtCents, CostCents: cur.CostCents,
				Stock: cur.Stock, WeightGrams: cur.WeightGrams, Status: cur.Status, Description: cur.Description,
			}
		}
		in.SKU = sku
		if v, ok := get("name"); ok {
			in.Name = v
		}
		if v, ok := get("category"); ok {
			in.Category = d.Category(matchCategory(v))
		}
		if v, ok := get("vendor"); ok {
			in.Vendor = v
		}
		if v, ok := get("tags"); ok {
			in.Tags = splitList(v)
		}
		if v, ok := get("status"); ok && v != "" {
			in.Status = d.ProductStatus(strings.ToLower(v))
		}
		if v, ok := get("description"); ok {
			in.Description = v
		}
		for _, m := range []struct {
			name string
			dst  *int64
		}{{"price", &in.PriceCents}, {"compare_at_price", &in.CompareAtCents}, {"cost", &in.CostCents}} {
			if v, ok := get(m.name); ok {
				c, err := parseDollars(v)
				if err != nil {
					fail(m.name, err.Error())
					continue
				}
				*m.dst = c
			}
		}
		for _, m := range []struct {
			name string
			dst  *int
		}{{"stock", &in.Stock}, {"weight_grams", &in.WeightGrams}} {
			if v, ok := get(m.name); ok && v != "" {
				n, err := strconv.Atoi(v)
				if err != nil {
					fail(m.name, "must be a whole number")
					continue
				}
				*m.dst = n
			}
		}

		in.Normalize()
		var verr *d.ValidationError
		if err := in.Validate(); errors.As(err, &verr) {
			for f, msg := range verr.Fields {
				fail(csvField(f), msg)
			}
			continue
		}
		plan = append(plan, planned{id: cur.ID, in: in})
	}

	if len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": fmt.Sprintf("%d problem(s) in the file — nothing was imported", len(errs)), "rows": errs,
		})
		return
	}

	res := importResult{Errors: []importRowError{}}
	for _, p := range plan {
		if p.id != 0 {
			_, err = s.store.UpdateProduct(r.Context(), p.id, p.in)
			res.Updated++
		} else {
			_, err = s.store.CreateProduct(r.Context(), p.in)
			res.Created++
		}
		if err != nil {
			s.writeDomainError(w, r, err)
			return
		}
	}
	me, _ := CurrentMember(r.Context())
	s.log.InfoContext(r.Context(), "products imported", "created", res.Created, "updated", res.Updated, "actor", me.Email)
	s.audit(r.Context(), d.ActImport, "", 0, "imported %d products from CSV (%d created, %d updated)", res.Created+res.Updated, res.Created, res.Updated)
	writeJSON(w, http.StatusOK, res)
}

func readImportBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes+64<<10)
	var src io.Reader = r.Body
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		f, _, err := r.FormFile("file")
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "CSV must be at most 2 MB")
			} else {
				writeError(w, http.StatusBadRequest, `expected a multipart form with a "file" field`)
			}
			return nil, false
		}
		defer f.Close()
		src = f
	}
	data, err := io.ReadAll(io.LimitReader(src, maxImportBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the upload")
		return nil, false
	}
	if len(data) > maxImportBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "CSV must be at most 2 MB")
		return nil, false
	}
	return data, true
}

// matchCategory accepts categories in any letter case.
func matchCategory(v string) string {
	for _, c := range d.Categories {
		if strings.EqualFold(string(c), v) {
			return string(c)
		}
	}
	return v
}

// csvField maps ProductInput json names back to CSV column names.
func csvField(f string) string {
	switch f {
	case "priceCents":
		return "price"
	case "compareAtCents":
		return "compare_at_price"
	case "costCents":
		return "cost"
	case "weightGrams":
		return "weight_grams"
	}
	return f
}
