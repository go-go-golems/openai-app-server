package harness

import "context"

type NotificationHandler func(ctx context.Context, evt NotificationEvent) error

type RequestHandler func(rc *RequestContext) error

type NotificationBinding struct {
	Method  string
	Handler NotificationHandler
}

type RequestBinding struct {
	Method  string
	Handler RequestHandler
}

type Harness struct {
	Name          string
	Notifications []NotificationBinding
	Requests      []RequestBinding
}

func Compose(harnesses ...Harness) *Dispatcher {
	d := NewDispatcher()
	for _, h := range harnesses {
		for _, n := range h.Notifications {
			d.OnNotification(n.Method, n.Handler)
		}
		for _, r := range h.Requests {
			d.OnRequest(r.Method, r.Handler)
		}
	}
	return d
}
