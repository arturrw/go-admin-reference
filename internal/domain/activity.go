package domain

import (
	"strconv"
	"time"
)

// USD formats cents for log messages: 1234567 → "$12,345.67".
func USD(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := strconv.FormatInt(cents/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	frac := strconv.FormatInt(cents%100+100, 10)[1:]
	return sign + "$" + whole + "." + frac
}

// Activity is one entry of the staff audit log: who did what, to which
// record. The dashboard shows the latest entries; the activity page all of them.
type Activity struct {
	ID       int64     `json:"id"`
	Kind     string    `json:"kind"`    // see ActivityKinds
	ActorID  int64     `json:"actorId"` // 0 = system (CI, inventory alerts)
	Actor    string    `json:"actor"`
	Message  string    `json:"message"` // reads after the actor: "published Aero Buds"
	Entity   string    `json:"entity"`  // product | order | customer | member | "" — what a click opens
	EntityID int64     `json:"entityId"`
	At       time.Time `json:"at"`
}

// Activity kinds, used for icons and filtering.
const (
	ActProduct  = "product"  // created, edited, deleted
	ActPublish  = "publish"  // product went live
	ActImage    = "image"    // gallery changes
	ActImport   = "import"   // CSV import
	ActOrder    = "order"    // status changes
	ActRefund   = "refund"   // refunds, with the reason
	ActNote     = "note"     // customer notes added or deleted
	ActTeam     = "team"     // invites, removals, profile edits
	ActRole     = "role"     // role and permission changes
	ActTarget   = "target"   // quarterly target
	ActSettings = "settings" // runtime settings
	ActAuth     = "auth"     // sign-ins (kept out of the dashboard feed)
	ActDeploy   = "deploy"   // system: releases
	ActStock    = "stock"    // system: inventory alerts
)

var ActivityKinds = []string{
	ActProduct, ActPublish, ActImage, ActImport, ActOrder, ActRefund, ActNote,
	ActTeam, ActRole, ActTarget, ActSettings, ActAuth, ActDeploy, ActStock,
}

// DashboardActivity is how many entries the dashboard feed shows.
const DashboardActivity = 6

type ActivityFilter struct {
	ActorID int64
	Kind    string
	Query   string // actor or message
	// ExcludeAuth hides sign-ins, which would drown out the dashboard feed.
	ExcludeAuth bool
	Limit       int
	Offset      int
}
