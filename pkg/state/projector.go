package state

import "strings"

type Projector struct {
	store *Store
}

func NewProjector(store *Store) *Projector {
	return &Projector{store: store}
}

func (p *Projector) Apply(method string, params any) {
	if p == nil || p.store == nil {
		return
	}
	m := asMap(params)
	if m == nil {
		return
	}

	switch method {
	case "thread/started", "thread/updated":
		p.applyThreadEvent(m)
	case "thread/tokenUsage/updated":
		p.applyTokenUsageEvent(m)
	case "turn/started", "turn/completed", "turn/updated":
		p.applyTurnEvent(m)
	case "turn/diff/updated":
		p.applyTurnDiffEvent(m)
	case "turn/plan/updated":
		p.applyTurnPlanEvent(m)
	case "item/started", "item/completed", "item/updated":
		p.applyItemEvent(method, m)
	default:
		return
	}
}

func (p *Projector) applyThreadEvent(params map[string]any) {
	thread := asMap(params["thread"])
	if thread == nil {
		thread = params
	}
	threadID := asString(thread["id"])
	if threadID == "" {
		return
	}

	p.store.UpsertThread(ThreadState{
		ID:     threadID,
		Status: asString(thread["status"]),
		Cwd:    asString(thread["cwd"]),
		Model:  asString(thread["model"]),
	})
}

func (p *Projector) applyTokenUsageEvent(params map[string]any) {
	threadID := asString(params["threadId"])
	if threadID == "" {
		return
	}
	tokenUsage := asMap(params["tokenUsage"])
	if tokenUsage != nil {
		p.store.SetThreadTokenUsage(threadID, tokenUsage)
	}
}

func (p *Projector) applyTurnEvent(params map[string]any) {
	threadID := asString(params["threadId"])
	turn := asMap(params["turn"])
	if turn == nil {
		turn = params
	}
	turnID := asString(turn["id"])
	if turnID == "" {
		turnID = asString(params["turnId"])
	}
	if threadID == "" || turnID == "" {
		return
	}

	p.store.UpsertTurn(threadID, TurnState{
		ID:       turnID,
		ThreadID: threadID,
		Status:   asString(turn["status"]),
		Error:    turn["error"],
	})
}

func (p *Projector) applyTurnDiffEvent(params map[string]any) {
	threadID := asString(params["threadId"])
	turnID := asString(params["turnId"])
	if threadID == "" || turnID == "" {
		return
	}
	p.store.SetTurnDiff(threadID, turnID, asString(params["diff"]))
}

func (p *Projector) applyTurnPlanEvent(params map[string]any) {
	threadID := asString(params["threadId"])
	turnID := asString(params["turnId"])
	if threadID == "" || turnID == "" {
		return
	}
	p.store.SetTurnPlan(threadID, turnID, params["plan"])
}

func (p *Projector) applyItemEvent(method string, params map[string]any) {
	threadID := asString(params["threadId"])
	turnID := asString(params["turnId"])
	item := asMap(params["item"])
	if threadID == "" || turnID == "" || item == nil {
		return
	}
	itemID := asString(item["id"])
	if itemID == "" {
		return
	}

	status := asString(item["status"])
	if status == "" {
		switch method {
		case "item/started":
			status = "inProgress"
		case "item/completed":
			status = "completed"
		}
	}

	p.store.UpsertItem(threadID, turnID, ItemState{
		ID:       itemID,
		ThreadID: threadID,
		TurnID:   turnID,
		Type:     asString(item["type"]),
		Status:   status,
		Raw:      item,
	})
}

func asMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func asString(v any) string {
	s, ok := v.(string)
	if ok {
		return strings.TrimSpace(s)
	}
	return ""
}
