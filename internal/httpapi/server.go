// Package httpapi exposes the admin JSON API under /api/v1 and serves the
// embedded single-page app for every other path.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/reqlog"
)

// Store is everything the API needs from persistence. The in-memory store
// implements it today; a Postgres implementation can be swapped in later.
type Store interface {
	ListProducts(ctx context.Context, f domain.ProductFilter) ([]domain.Product, error)
	ProductStats(ctx context.Context) (domain.ProductStats, error)
	GetProduct(ctx context.Context, id int64) (domain.Product, error)
	CreateProduct(ctx context.Context, in domain.ProductInput) (domain.Product, error)
	UpdateProduct(ctx context.Context, id int64, in domain.ProductInput) (domain.Product, error)
	DeleteProduct(ctx context.Context, id int64) error
	BulkProducts(ctx context.Context, ids []int64, action domain.BulkAction) (int, error)

	ListOrders(ctx context.Context, f domain.OrderFilter) ([]domain.Order, error)
	OrderCounts(ctx context.Context) (map[domain.OrderStatus]int, error)
	GetOrder(ctx context.Context, id int64) (domain.Order, error)
	UpdateOrderStatus(ctx context.Context, id int64, status domain.OrderStatus) (domain.Order, error)

	ListCustomers(ctx context.Context, f domain.CustomerFilter) ([]domain.Customer, error)
	CustomerSegments(ctx context.Context) (map[string]domain.SegmentSummary, error)

	ListMembers(ctx context.Context, role domain.Role) ([]domain.Member, error)
	CreateMember(ctx context.Context, in domain.MemberInput) (domain.Member, error)
	UpdateMember(ctx context.Context, id int64, in domain.MemberInput) (domain.Member, error)
	DeleteMember(ctx context.Context, id int64) error

	Dashboard(ctx context.Context, days int) (domain.Dashboard, error)
}

type Deps struct {
	Logger    *slog.Logger
	Store     Store
	Requests  *reqlog.Log
	SPA       http.Handler // serves the built frontend
	Version   string
	Env       string
	StartedAt time.Time
}

type server struct {
	log      *slog.Logger
	store    Store
	requests *reqlog.Log
	version  string
	env      string
	started  time.Time
}

func New(deps Deps) http.Handler {
	s := &server{
		log: deps.Logger, store: deps.Store, requests: deps.Requests,
		version: deps.Version, env: deps.Env, started: deps.StartedAt,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/v1/meta", s.meta)
	mux.HandleFunc("GET /api/v1/runtime", s.runtime)
	mux.HandleFunc("GET /api/v1/dashboard", s.dashboard)

	mux.HandleFunc("GET /api/v1/products", s.listProducts)
	mux.HandleFunc("POST /api/v1/products", s.createProduct)
	mux.HandleFunc("POST /api/v1/products/bulk", s.bulkProducts)
	mux.HandleFunc("GET /api/v1/products/{id}", s.getProduct)
	mux.HandleFunc("PUT /api/v1/products/{id}", s.updateProduct)
	mux.HandleFunc("DELETE /api/v1/products/{id}", s.deleteProduct)

	mux.HandleFunc("GET /api/v1/orders", s.listOrders)
	mux.HandleFunc("GET /api/v1/orders/{id}", s.getOrder)
	mux.HandleFunc("PATCH /api/v1/orders/{id}/status", s.updateOrderStatus)

	mux.HandleFunc("GET /api/v1/customers", s.listCustomers)

	mux.HandleFunc("GET /api/v1/team", s.listMembers)
	mux.HandleFunc("POST /api/v1/team", s.createMember)
	mux.HandleFunc("PUT /api/v1/team/{id}", s.updateMember)
	mux.HandleFunc("DELETE /api/v1/team/{id}", s.deleteMember)

	mux.HandleFunc("GET /api/v1/requests", s.listRequests)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "no such endpoint")
	})
	if deps.SPA != nil {
		mux.Handle("/", deps.SPA)
	}

	// Outermost first: request ID → access log → panic recovery → headers → mux.
	return withRequestID(s.withAccessLog(s.withRecover(securityHeaders(mux))))
}
