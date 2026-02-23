package state

import (
	"encoding/json"
	"sync"
)

type Config struct {
	MaxThreads        int
	MaxTurnsPerThread int
	MaxItemsPerTurn   int
}

type Store struct {
	mu sync.RWMutex

	cfg Config

	threads     map[string]*threadRecord
	threadOrder []string
}

type threadRecord struct {
	state     ThreadState
	turns     map[string]*turnRecord
	turnOrder []string
}

type turnRecord struct {
	state     TurnState
	items     map[string]*ItemState
	itemOrder []string
}

func NewStore(cfg Config) *Store {
	cfg = normalizeConfig(cfg)
	return &Store{
		cfg:     cfg,
		threads: map[string]*threadRecord{},
	}
}

func normalizeConfig(cfg Config) Config {
	if cfg.MaxThreads <= 0 {
		cfg.MaxThreads = 100
	}
	if cfg.MaxTurnsPerThread <= 0 {
		cfg.MaxTurnsPerThread = 100
	}
	if cfg.MaxItemsPerTurn <= 0 {
		cfg.MaxItemsPerTurn = 200
	}
	return cfg
}

func (s *Store) UpsertThread(thread ThreadState) {
	if thread.ID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec := s.ensureThreadLocked(thread.ID)
	if thread.Status != "" {
		rec.state.Status = thread.Status
	}
	if thread.Cwd != "" {
		rec.state.Cwd = thread.Cwd
	}
	if thread.Model != "" {
		rec.state.Model = thread.Model
	}
	if thread.TokenUsage != nil {
		rec.state.TokenUsage = cloneMap(thread.TokenUsage)
	}
}

func (s *Store) UpsertTurn(threadID string, turn TurnState) {
	if threadID == "" || turn.ID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tRec := s.ensureThreadLocked(threadID)
	turnRec := s.ensureTurnLocked(tRec, threadID, turn.ID)

	turnRec.state.ThreadID = threadID
	if turn.Status != "" {
		turnRec.state.Status = turn.Status
	}
	if turn.Error != nil {
		turnRec.state.Error = cloneAny(turn.Error)
	}
	if turn.LatestDiff != "" {
		turnRec.state.LatestDiff = turn.LatestDiff
	}
	if turn.LatestPlan != nil {
		turnRec.state.LatestPlan = cloneAny(turn.LatestPlan)
	}
}

func (s *Store) UpsertItem(threadID string, turnID string, item ItemState) {
	if threadID == "" || turnID == "" || item.ID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tRec := s.ensureThreadLocked(threadID)
	turnRec := s.ensureTurnLocked(tRec, threadID, turnID)
	itemRec, exists := turnRec.items[item.ID]
	if !exists {
		itemCopy := &ItemState{ID: item.ID}
		turnRec.items[item.ID] = itemCopy
		turnRec.itemOrder = append(turnRec.itemOrder, item.ID)
		itemRec = itemCopy
		s.enforceItemBoundLocked(turnRec)
	}

	itemRec.ThreadID = threadID
	itemRec.TurnID = turnID
	if item.Type != "" {
		itemRec.Type = item.Type
	}
	if item.Status != "" {
		itemRec.Status = item.Status
	}
	if item.Raw != nil {
		itemRec.Raw = cloneMap(item.Raw)
	}
}

func (s *Store) SetTurnDiff(threadID string, turnID string, diff string) {
	if threadID == "" || turnID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tRec := s.ensureThreadLocked(threadID)
	turnRec := s.ensureTurnLocked(tRec, threadID, turnID)
	turnRec.state.LatestDiff = diff
}

func (s *Store) SetTurnPlan(threadID string, turnID string, plan any) {
	if threadID == "" || turnID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tRec := s.ensureThreadLocked(threadID)
	turnRec := s.ensureTurnLocked(tRec, threadID, turnID)
	turnRec.state.LatestPlan = cloneAny(plan)
}

func (s *Store) SetThreadTokenUsage(threadID string, tokenUsage map[string]any) {
	if threadID == "" || tokenUsage == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tRec := s.ensureThreadLocked(threadID)
	tRec.state.TokenUsage = cloneMap(tokenUsage)
}

func (s *Store) Thread(threadID string) (ThreadState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tRec, ok := s.threads[threadID]
	if !ok {
		return ThreadState{}, false
	}
	return snapshotThread(tRec), true
}

func (s *Store) Turn(threadID string, turnID string) (TurnState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tRec, ok := s.threads[threadID]
	if !ok {
		return TurnState{}, false
	}
	turnRec, ok := tRec.turns[turnID]
	if !ok {
		return TurnState{}, false
	}
	return snapshotTurn(turnRec), true
}

func (s *Store) ListThreads() []ThreadState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ThreadState, 0, len(s.threadOrder))
	for _, threadID := range s.threadOrder {
		tRec, ok := s.threads[threadID]
		if !ok {
			continue
		}
		out = append(out, snapshotThread(tRec))
	}
	return out
}

func (s *Store) ensureThreadLocked(threadID string) *threadRecord {
	rec, ok := s.threads[threadID]
	if ok {
		return rec
	}

	rec = &threadRecord{
		state: ThreadState{ID: threadID},
		turns: map[string]*turnRecord{},
	}
	s.threads[threadID] = rec
	s.threadOrder = append(s.threadOrder, threadID)
	s.enforceThreadBoundLocked()
	return rec
}

func (s *Store) ensureTurnLocked(tRec *threadRecord, threadID string, turnID string) *turnRecord {
	turnRec, ok := tRec.turns[turnID]
	if ok {
		return turnRec
	}

	turnRec = &turnRecord{
		state: TurnState{ID: turnID, ThreadID: threadID},
		items: map[string]*ItemState{},
	}
	tRec.turns[turnID] = turnRec
	tRec.turnOrder = append(tRec.turnOrder, turnID)
	s.enforceTurnBoundLocked(tRec)
	return turnRec
}

func (s *Store) enforceThreadBoundLocked() {
	for len(s.threadOrder) > s.cfg.MaxThreads {
		evictID := s.threadOrder[0]
		s.threadOrder = s.threadOrder[1:]
		delete(s.threads, evictID)
	}
}

func (s *Store) enforceTurnBoundLocked(tRec *threadRecord) {
	for len(tRec.turnOrder) > s.cfg.MaxTurnsPerThread {
		evictID := tRec.turnOrder[0]
		tRec.turnOrder = tRec.turnOrder[1:]
		delete(tRec.turns, evictID)
	}
}

func (s *Store) enforceItemBoundLocked(turnRec *turnRecord) {
	for len(turnRec.itemOrder) > s.cfg.MaxItemsPerTurn {
		evictID := turnRec.itemOrder[0]
		turnRec.itemOrder = turnRec.itemOrder[1:]
		delete(turnRec.items, evictID)
	}
}

func snapshotThread(tRec *threadRecord) ThreadState {
	out := ThreadState{
		ID:     tRec.state.ID,
		Status: tRec.state.Status,
		Cwd:    tRec.state.Cwd,
		Model:  tRec.state.Model,
	}
	if tRec.state.TokenUsage != nil {
		out.TokenUsage = cloneMap(tRec.state.TokenUsage)
	}
	for _, turnID := range tRec.turnOrder {
		turnRec, ok := tRec.turns[turnID]
		if !ok {
			continue
		}
		turn := snapshotTurn(turnRec)
		out.Turns = append(out.Turns, &turn)
	}
	return out
}

func snapshotTurn(turnRec *turnRecord) TurnState {
	out := TurnState{
		ID:         turnRec.state.ID,
		ThreadID:   turnRec.state.ThreadID,
		Status:     turnRec.state.Status,
		Error:      cloneAny(turnRec.state.Error),
		LatestDiff: turnRec.state.LatestDiff,
		LatestPlan: cloneAny(turnRec.state.LatestPlan),
	}
	for _, itemID := range turnRec.itemOrder {
		itemRec, ok := turnRec.items[itemID]
		if !ok {
			continue
		}
		item := &ItemState{
			ID:       itemRec.ID,
			ThreadID: itemRec.ThreadID,
			TurnID:   itemRec.TurnID,
			Type:     itemRec.Type,
			Status:   itemRec.Status,
		}
		if itemRec.Raw != nil {
			item.Raw = cloneMap(itemRec.Raw)
		}
		out.Items = append(out.Items, item)
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneAny(v)
	}
	return out
}

func cloneAny(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}
