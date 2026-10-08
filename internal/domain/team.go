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
	// Exceptions to the role, set by the owner (see Permissions).
	Granted []Permission `json:"granted"`
	Revoked []Permission `json:"revoked"`
	// NotificationsReadAt is when the member last opened their notifications.
	NotificationsReadAt *time.Time `json:"-"`
	// A pending invitation: the SHA-256 of its token and when it lapses.
	InviteHash      []byte     `json:"-"`
	InviteExpiresAt *time.Time `json:"inviteExpiresAt"`
}

// InviteLifetime is how long an invitation link works.
const InviteLifetime = 7 * 24 * time.Hour

// MinPasswordLength applies when an invitee chooses a password.
const MinPasswordLength = 8

// ValidateNewPassword checks a password chosen on accepting an invitation.
func ValidateNewPassword(password, email string) error {
	v := validator{}
	v.check(len(password) >= MinPasswordLength, "password", "must be at least 8 characters")
	v.check(len(password) <= 128, "password", "must be at most 128 characters")
	v.check(!strings.EqualFold(password, email), "password", "must not be your email address")
	return v.err()
}

// OnlineWithin is how recently a member must have made a request to count as online.
const OnlineWithin = 5 * time.Minute

// Permissions is what the member may do: the role's permissions plus granted
// minus revoked, in display order. The owner always has everything.
func (m Member) Permissions() []Permission {
	role := RolePermissions[m.Role]
	out := []Permission{}
	for _, p := range Permissions {
		has := slices.Contains(role, p.Key)
		if m.Role != RoleOwner {
			has = (has || slices.Contains(m.Granted, p.Key)) && !slices.Contains(m.Revoked, p.Key)
		}
		if has {
			out = append(out, p.Key)
		}
	}
	return out
}

func (m Member) Can(p Permission) bool { return slices.Contains(m.Permissions(), p) }

// MemberAccess is a request to change a member's exceptions.
type MemberAccess struct {
	Granted []Permission `json:"granted"`
	Revoked []Permission `json:"revoked"`
}

// Normalize validates the exceptions against the member's role and drops the
// ones that change nothing (granting what the role has, revoking what it lacks).
func (a MemberAccess) Normalize(role Role) (MemberAccess, error) {
	known := func(p Permission) bool {
		return slices.ContainsFunc(Permissions, func(i PermissionInfo) bool { return i.Key == p })
	}
	v := validator{}
	out := MemberAccess{Granted: []Permission{}, Revoked: []Permission{}}
	for _, p := range a.Granted {
		v.check(known(p), "granted", "unknown permission "+string(p))
		v.check(p != PermWorkspaceManage, "granted", "the danger zone stays with the owner")
		if known(p) && !role.Can(p) && !slices.Contains(out.Granted, p) {
			out.Granted = append(out.Granted, p)
		}
	}
	for _, p := range a.Revoked {
		v.check(known(p), "revoked", "unknown permission "+string(p))
		v.check(!slices.Contains(a.Granted, p), "revoked", "cannot grant and revoke "+string(p))
		if known(p) && role.Can(p) && !slices.Contains(out.Revoked, p) {
			out.Revoked = append(out.Revoked, p)
		}
	}
	return out, v.err()
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
	{PermWorkspaceManage, "System", "Danger zone", "Clear the request log, sign everyone out, set the quarterly target"},
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
