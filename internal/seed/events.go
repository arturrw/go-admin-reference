package seed

import (
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// shippers pack orders in the demo data; the same person appears in the
// activity feed, because both come from these events.
var shippers = []string{"Mark Liu", "Omar Haddad", "Yuki Tanaka"}

// ShippedAt is when the demo warehouse sent an order out.
func shippedAt(o d.Order) time.Time {
	return o.PlacedAt.Add(5*time.Hour + time.Duration(o.ID%50)*time.Minute)
}

// OrderEvents writes the history an order in the demo data would have had,
// from its status: placed, paid a minute later, shipped the same day,
// delivered two days after that. Nothing is in the future.
func OrderEvents(o d.Order, now time.Time) []d.OrderEvent {
	at := func(t time.Time) time.Time {
		if t.After(now) {
			return now
		}
		return t
	}
	ev := []d.OrderEvent{{Status: d.OrderPending, At: o.PlacedAt, By: "Customer"}}
	add := func(s d.OrderStatus, t time.Time, by string) {
		ev = append(ev, d.OrderEvent{Status: s, At: at(t), By: by})
	}
	paid := o.PlacedAt.Add(time.Minute)
	switch o.Status {
	case d.OrderPending:
	case d.OrderFailed:
		add(d.OrderFailed, paid, "Payment provider")
	default:
		add(d.OrderPaid, paid, "Payment provider")
		shipped := shippedAt(o)
		shipBy := shippers[int(o.ID)%len(shippers)]
		switch o.Status {
		case d.OrderShipped:
			add(d.OrderShipped, shipped, shipBy)
		case d.OrderDelivered:
			add(d.OrderShipped, shipped, shipBy)
			add(d.OrderDelivered, shipped.Add(48*time.Hour+time.Duration(o.ID%6)*time.Hour), "Carrier")
		case d.OrderRefunded:
			if o.Refund != nil && shipped.Before(o.Refund.At) {
				add(d.OrderShipped, shipped, shipBy)
			}
			if o.Refund != nil {
				add(d.OrderRefunded, o.Refund.At, o.Refund.By)
			}
		}
	}
	return ev
}
