package domain

import (
	"net/mail"
	"slices"
	"strings"
	"time"
)

type Role string

const (
	RoleOwner   Role = "owner"
	RoleAdmin   Role = "admin"
	RoleEditor  Role = "editor"
	RoleSupport Role = "support"
	RoleViewer  Role = "viewer"
)

var Roles = []Role{RoleOwner, RoleAdmin, RoleEditor, RoleSupport, RoleViewer}

type MemberStatus string

const (
	MemberActive    MemberStatus = "active"
	MemberInvited   MemberStatus = "invited"
	MemberSuspended MemberStatus = "suspended"
)

type Member struct {
	ID           int64        `json:"id"`
	Name         string       `json:"name"`
	Email        string       `json:"email"`
	Role         Role         `json:"role"`
	Status       MemberStatus `json:"status"`
	MFA          bool         `json:"mfa"`
	LastActiveAt *time.Time   `json:"lastActiveAt"`
	PasswordHash string       `json:"-"`
}

type MemberInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  Role   `json:"role"`
}

func (in *MemberInput) Normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
}

func (in MemberInput) Validate() error {
	v := validator{}
	v.check(in.Name != "", "name", "is required")
	_, err := mail.ParseAddress(in.Email)
	v.check(err == nil, "email", "must be a valid email address")
	v.check(slices.Contains([]Role{RoleAdmin, RoleEditor, RoleSupport, RoleViewer}, in.Role), "role", "must be admin, editor, support or viewer")
	return v.err()
}

// ── Permissions ─────────────────────────────────────────────────────────────

type Permission string

const (
	PermDashboard       Permission = "dashboard:read"
	PermProductsRead    Permission = "products:read"
	PermProductsWrite   Permission = "products:write"
	PermOrdersRead      Permission = "orders:read"
	PermOrdersWrite     Permission = "orders:write"
	PermCustomersRead   Permission = "customers:read"
	PermCustomersWrite  Permission = "customers:write"
	PermTeamRead        Permission = "team:read"
	PermTeamWrite       Permission = "team:write"
	PermRequestsRead    Permission = "requests:read"
	PermSettingsWrite   Permission = "settings:write"
	PermWorkspaceManage Permission = "workspace:manage"
)

type PermissionInfo struct {
	Key         Permission `json:"key"`
	Group       string     `json:"group"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
}

// Permissions lists every permission in display order.
var Permissions = []PermissionInfo{
	{PermDashboard, "Overview", "View dashboard", "Revenue, KPIs, runtime stats"},
	{PermProductsRead, "Products", "View products", "Catalogue, stock and pricing"},
	{PermProductsWrite, "Products", "Edit products", "Create, edit, delete, upload images, bulk actions"},
	{PermOrdersRead, "Orders", "View orders", "Order list and details"},
	{PermOrdersWrite, "Orders", "Manage orders", "Change status, ship, refund"},
	{PermCustomersRead, "Customers", "View customers", "Profiles, purchase history"},
	{PermCustomersWrite, "Customers", "Annotate customers", "Add internal notes"},
	{PermTeamRead, "Team", "View team", "Members and their roles"},
	{PermTeamWrite, "Team", "Manage team", "Invite, edit and remove members"},
	{PermRequestsRead, "System", "Request log", "Inspect API traffic, headers and bodies"},
	{PermSettingsWrite, "System", "Edit settings", "Service, security and API keys"},
	{PermWorkspaceManage, "System", "Danger zone", "Flush cache, delete workspace"},
}

// RolePermissions is the access matrix enforced by the API and mirrored in the UI.
var RolePermissions = map[Role][]Permission{
	RoleOwner: {
		PermDashboard, PermProductsRead, PermProductsWrite, PermOrdersRead, PermOrdersWrite,
		PermCustomersRead, PermCustomersWrite, PermTeamRead, PermTeamWrite, PermRequestsRead,
		PermSettingsWrite, PermWorkspaceManage,
	},
	RoleAdmin: {
		PermDashboard, PermProductsRead, PermProductsWrite, PermOrdersRead, PermOrdersWrite,
		PermCustomersRead, PermCustomersWrite, PermTeamRead, PermTeamWrite, PermRequestsRead,
		PermSettingsWrite,
	},
	RoleEditor: {
		PermDashboard, PermProductsRead, PermProductsWrite, PermOrdersRead, PermOrdersWrite,
		PermCustomersRead, PermTeamRead,
	},
	RoleSupport: {
		PermDashboard, PermProductsRead, PermOrdersRead, PermOrdersWrite,
		PermCustomersRead, PermCustomersWrite,
	},
	RoleViewer: {
		PermDashboard, PermProductsRead, PermOrdersRead, PermCustomersRead,
	},
}

func (r Role) Can(p Permission) bool { return slices.Contains(RolePermissions[r], p) }
