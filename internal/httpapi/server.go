// Package httpapi exposes the admin JSON API under /api/v1 and serves the
// embedded single-page app for every other path.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/media"
	"github.com/arturrw/go-admin-reference/internal/reqlog"
)

// Store is everything the API needs from persistence. The in-memory store
// implements it today; a Postgres implementation can be swapped in later.
type Store interface {
	ListProducts(ctx context.Context, f d.ProductFilter) ([]d.Product, error)
	ProductStats(ctx context.Context) (d.ProductStats, error)
	GetProduct(ctx context.Context, id int64) (d.Product, error)
	CreateProduct(ctx context.Context, in d.ProductInput) (d.Product, error)
	UpdateProduct(ctx context.Context, id int64, in d.ProductInput) (d.Product, error)
	DeleteProduct(ctx context.Context, id int64) (d.Product, error)
	BulkProducts(ctx context.Context, ids []int64, action d.BulkAction) ([]d.Product, error)
	AddProductImage(ctx context.Context, productID int64, img d.ProductImage) (d.Product, error)
	DeleteProductImage(ctx context.Context, productID int64, imageID string) (d.Product, d.ProductImage, error)
	SetPrimaryImage(ctx context.Context, productID int64, imageID string) (d.Product, error)

	ListOrders(ctx context.Context, f d.OrderFilter) ([]d.Order, int, error)
	OrderCounts(ctx context.Context) (map[d.OrderStatus]int, error)
	GetOrder(ctx context.Context, id int64) (d.Order, error)
	// CreateOrder records a sale at current catalogue prices and takes the
	// stock; by is who entered it. Unknown customer or product is a validation
	// error, too little stock a ConflictError.
	CreateOrder(ctx context.Context, in d.NewOrder, by string) (d.Order, error)
	// EditOrderItems replaces the lines of a pending order, moving stock by the
	// difference; any other status is a ConflictError.
	EditOrderItems(ctx context.Context, id int64, items []d.NewOrderLine) (d.Order, error)
	// UpdateOrderStatus sets the status; refund is recorded with a refund and nil otherwise.
	UpdateOrderStatus(ctx context.Context, id int64, status d.OrderStatus, refund *d.OrderRefund, by string) (d.Order, error)

	ListCustomers(ctx context.Context, f d.CustomerFilter) ([]d.Customer, error)
	CustomerSegments(ctx context.Context) (map[string]d.SegmentSummary, error)
	GetCustomer(ctx context.Context, id int64) (d.CustomerDetail, error)
	CreateCustomer(ctx context.Context, in d.CustomerInput) (d.Customer, error)
	AddCustomerNote(ctx context.Context, customerID int64, author, text string) (d.CustomerNote, error)
	DeleteCustomerNote(ctx context.Context, customerID, noteID int64) (d.CustomerNote, error)

	ListMembers(ctx context.Context, role d.Role) ([]d.Member, error)
	GetMember(ctx context.Context, id int64) (d.Member, error)
	MemberByEmail(ctx context.Context, email string) (d.Member, error)
	TouchMember(ctx context.Context, id int64)
	MarkNotificationsRead(ctx context.Context, id int64, at time.Time) error
	// SetInvite replaces the pending invitation of an invited member.
	SetInvite(ctx context.Context, id int64, hash []byte, expires time.Time) error
	MemberByInvite(ctx context.Context, hash []byte) (d.Member, error)
	AcceptInvite(ctx context.Context, id int64, name, passwordHash string) (d.Member, error)
	CreateMember(ctx context.Context, in d.MemberInput) (d.Member, error)
	UpdateMember(ctx context.Context, id int64, in d.MemberInput) (d.Member, error)
	DeleteMember(ctx context.Context, id int64) error
	SetMemberAccess(ctx context.Context, id int64, a d.MemberAccess) (d.Member, error)
	SetMemberStatus(ctx context.Context, id int64, status d.MemberStatus) (d.Member, error)

	GetSettings(ctx context.Context) (d.Settings, error)
	SaveSettings(ctx context.Context, s d.Settings) error

	ListAPIKeys(ctx context.Context) ([]d.APIKey, error)
	CreateAPIKey(ctx context.Context, k d.APIKey) (d.APIKey, error)
	APIKeyByHash(ctx context.Context, hash []byte) (d.APIKey, error)
	TouchAPIKey(ctx context.Context, id int64)
	RevokeAPIKey(ctx context.Context, id int64) (d.APIKey, error)

	RecordActivity(ctx context.Context, a d.Activity) (d.Activity, error)
	ListActivity(ctx context.Context, f d.ActivityFilter) ([]d.Activity, int, error)

	// TargetGoal returns the quarter's stored goal, or nil while it is unset.
	TargetGoal(ctx context.Context, quarter string) (*d.TargetGoal, error)
	SetTargetGoal(ctx context.Context, g d.TargetGoal) (d.TargetGoal, error)

	Dashboard(ctx context.Context, days int) (d.Dashboard, error)
}

// SessionStore persists login sessions (auth.MemorySessions or postgres.Sessions).
type SessionStore interface {
	Create(ctx context.Context, memberID int64) (token string, err error)
	Lookup(ctx context.Context, token string) (memberID int64, err error)
	Delete(ctx context.Context, token string)
	DeleteMember(ctx context.Context, memberID int64)
	// DeleteOthers ends every session except the one with token keep.
	DeleteOthers(ctx context.Context, keep string) int64
	TTL() time.Duration
	SetTTL(ttl time.Duration)
}

// Pinger is implemented by stores backed by a database, for /healthz.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	Logger    *slog.Logger
	LogLevel  *slog.LevelVar // optional; enables GET/PUT /api/v1/settings/log-level
	Store     Store
	Requests  *reqlog.Log
	Sessions  SessionStore
	Media     *media.Storage
	SPA       http.Handler // serves the built frontend
	Version   string
	Env       string
	StartedAt time.Time
	Addr      string // listen address, shown read-only in Settings
	// WebhookBackoff is the wait before the 2nd and 3rd delivery attempt
	// (default 1s, 5s); tests shorten it.
	WebhookBackoff []time.Duration
	// DemoPassword is shown on the login page in development; empty disables it.
	DemoPassword string
}

type server struct {
	log      *slog.Logger
	level    *slog.LevelVar
	store    Store
	requests *reqlog.Log
	sessions SessionStore
	media    *media.Storage
	version  string
	env      string
	started  time.Time
	addr     string
	cfg      settingsCache
	hooks    *webhookState

	demoPassword string
}

func (s *server) isDev() bool { return s.env != "production" }

func New(deps Deps) http.Handler {
	s := &server{
		log: deps.Logger, level: deps.LogLevel, store: deps.Store, requests: deps.Requests, sessions: deps.Sessions,
		media: deps.Media, version: deps.Version, env: deps.Env, started: deps.StartedAt, addr: deps.Addr, hooks: newWebhookState(deps.WebhookBackoff),
		demoPassword: deps.DemoPassword,
	}

	mux := http.NewServeMux()
	// route registers a handler guarded by a permission ("" = any signed-in member).
	route := func(pattern string, perm d.Permission, h http.HandlerFunc) {
		mux.Handle(pattern, s.authorize(perm, h))
	}

	mux.HandleFunc("GET /healthz", s.health)

	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/auth/demo-accounts", s.demoAccounts)
	mux.HandleFunc("GET /api/v1/auth/invite/{token}", s.inviteDetails)
	mux.HandleFunc("POST /api/v1/auth/invite/{token}", s.acceptInvite)
	route("GET /api/v1/auth/me", "", s.me)
	route("GET /api/v1/roles", "", s.roles)

	route("GET /api/v1/meta", "", s.meta)
	route("GET /api/v1/runtime", d.PermDashboard, s.runtime)
	route("GET /api/v1/live", d.PermDashboard, s.live)
	route("GET /api/v1/dashboard", d.PermDashboard, s.dashboard)
	route("PUT /api/v1/target", d.PermWorkspaceManage, s.setTarget)

	route("GET /api/v1/products", d.PermProductsRead, s.listProducts)
	route("POST /api/v1/products", d.PermProductsWrite, s.createProduct)
	route("POST /api/v1/products/bulk", d.PermProductsWrite, s.bulkProducts)
	route("GET /api/v1/products/export", d.PermProductsRead, s.exportProducts)
	route("POST /api/v1/products/import", d.PermProductsWrite, s.importProducts)
	route("GET /api/v1/products/{id}", d.PermProductsRead, s.getProduct)
	route("PUT /api/v1/products/{id}", d.PermProductsWrite, s.updateProduct)
	route("DELETE /api/v1/products/{id}", d.PermProductsWrite, s.deleteProduct)
	route("POST /api/v1/products/{id}/images", d.PermProductsWrite, s.uploadProductImage)
	route("DELETE /api/v1/products/{id}/images/{imageId}", d.PermProductsWrite, s.deleteProductImage)
	route("POST /api/v1/products/{id}/images/{imageId}/primary", d.PermProductsWrite, s.setPrimaryImage)

	route("GET /api/v1/orders", d.PermOrdersRead, s.listOrders)
	route("POST /api/v1/orders", d.PermOrdersWrite, s.createOrder)
	route("GET /api/v1/orders/export", d.PermOrdersRead, s.exportOrders)
	route("GET /api/v1/orders/{id}", d.PermOrdersRead, s.getOrder)
	route("PATCH /api/v1/orders/{id}/status", d.PermOrdersWrite, s.updateOrderStatus)
	route("PUT /api/v1/orders/{id}/items", d.PermOrdersWrite, s.editOrderItems)
	route("GET /api/v1/orders/{id}/invoice", d.PermOrdersRead, s.invoice)

	route("GET /api/v1/customers", d.PermCustomersRead, s.listCustomers)
	route("POST /api/v1/customers", d.PermCustomersWrite, s.createCustomer)
	route("GET /api/v1/customers/export", d.PermCustomersRead, s.exportCustomers)
	route("GET /api/v1/customers/{id}", d.PermCustomersRead, s.getCustomer)
	route("POST /api/v1/customers/{id}/notes", d.PermCustomersWrite, s.addCustomerNote)
	route("DELETE /api/v1/customers/{id}/notes/{noteId}", d.PermCustomersWrite, s.deleteCustomerNote)

	route("GET /api/v1/team", d.PermTeamRead, s.listMembers)
	route("POST /api/v1/team", d.PermTeamWrite, s.createMember)
	route("GET /api/v1/team/{id}", d.PermTeamRead, s.getMember)
	route("PUT /api/v1/team/{id}", d.PermTeamWrite, s.updateMember)
	route("PUT /api/v1/team/{id}/access", d.PermTeamWrite, s.setMemberAccess)
	route("POST /api/v1/team/{id}/invite", d.PermTeamWrite, s.resendInvite)
	route("PUT /api/v1/team/{id}/status", d.PermTeamWrite, s.setMemberStatus)
	route("DELETE /api/v1/team/{id}", d.PermTeamWrite, s.deleteMember)

	route("GET /api/v1/activity", d.PermTeamRead, s.listActivity)
	route("GET /api/v1/notifications", d.PermDashboard, s.notifications)
	route("POST /api/v1/notifications/read", d.PermDashboard, s.readNotifications)

	route("GET /api/v1/settings/api-keys", d.PermSettingsWrite, s.listAPIKeys)
	route("POST /api/v1/settings/api-keys", d.PermSettingsWrite, s.createAPIKey)
	route("DELETE /api/v1/settings/api-keys/{id}", d.PermSettingsWrite, s.revokeAPIKey)
	route("POST /api/v1/danger/clear-request-log", d.PermWorkspaceManage, s.clearRequestLog)
	route("POST /api/v1/danger/sign-out-everyone", d.PermWorkspaceManage, s.signOutEveryone)
	route("GET /api/v1/settings", d.PermSettingsWrite, s.getSettings)
	route("PATCH /api/v1/settings", d.PermSettingsWrite, s.patchSettings)
	route("GET /api/v1/settings/webhook/deliveries", d.PermSettingsWrite, s.listDeliveries)
	route("POST /api/v1/settings/webhook/test", d.PermSettingsWrite, s.testWebhook)
	route("POST /api/v1/settings/webhook/rotate-secret", d.PermSettingsWrite, s.rotateWebhookSecret)
	route("GET /api/v1/settings/log-level", "", s.getLogLevel)
	route("PUT /api/v1/settings/log-level", d.PermSettingsWrite, s.setLogLevel)

	route("GET /api/v1/requests", d.PermRequestsRead, s.listRequests)
	route("GET /api/v1/requests/{id}", d.PermRequestsRead, s.getRequest)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "no such endpoint")
	})

	// Product images are public, like a storefront CDN would serve them.
	mux.HandleFunc("GET /media/generated/{product}/{file}", s.generatedImage)
	if s.media != nil {
		mux.Handle("GET /media/uploads/", http.StripPrefix("/media/uploads/", s.media.Handler()))
	}
	if deps.SPA != nil {
		mux.Handle("/", deps.SPA)
	}

	// Outermost first: request ID → access log → panic recovery → headers → mux.
	return withRequestID(s.withAccessLog(s.withRecover(securityHeaders(mux))))
}
