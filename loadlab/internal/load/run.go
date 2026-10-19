package load

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Phase string

const (
	PhaseSeeding   Phase = "seeding"
	PhaseRunning   Phase = "running"
	PhaseDraining  Phase = "draining"
	PhaseVerifying Phase = "verifying"
	PhaseDone      Phase = "done"
	PhaseStopped   Phase = "stopped"
	PhaseFailed    Phase = "failed"
)

// Every seeded wallet starts with a million: no scenario can spend it in
// the longest run allowed, so a rejection for want of balance means a bug.
const initialBalance Cents = 100_000_000

const (
	betAmount Cents = 100
	winAmount Cents = 50
	// One arrival in ten is a win.
	winShare = 0.1
	// In the replay scenario, half of the arrivals resend a completed
	// operation.
	replayShare = 0.5
	// How many completed operations a replay picks from.
	replayPool = 1024
	// Seeding, lookups and verification run this many requests at once.
	setupWorkers = 32
	// The scheduler wakes this often and issues every arrival due since.
	tick = time.Millisecond
)

// Mismatch is a wallet whose balance differs from the load lab's books.
type Mismatch struct {
	WalletID string `json:"walletId"`
	Expected Cents  `json:"expectedCents"`
	Actual   Cents  `json:"actualCents"`
}

// Verification compares, after the load, every wallet's balance with what
// the answers the load lab got say it should be, and asks the API to
// reconcile each wallet with its ledger.
type Verification struct {
	Passed                      bool       `json:"passed"`
	WalletsChecked              int        `json:"walletsChecked"`
	Mismatches                  []Mismatch `json:"mismatches"`
	InconsistentReconciliations int        `json:"inconsistentReconciliations"`
	// Operations whose answer was lost, found afterwards by external ID.
	Resolved   int    `json:"resolved"`
	Unresolved int    `json:"unresolved"`
	Error      string `json:"error,omitempty"`
}

// Result is a finished run.
type Result struct {
	Phase           Phase            `json:"phase"`
	Offered         int64            `json:"offered"`
	Sent            int64            `json:"sent"`
	Completed       int64            `json:"completed"`
	Dropped         int64            `json:"dropped"`
	Errors          int64            `json:"errors"`
	Replays         int64            `json:"replays"`
	Statuses        map[string]int64 `json:"statuses"`
	LatencyMs       Latency          `json:"latencyMs"`
	LoadSeconds     float64          `json:"loadSeconds"`
	CompletedPerSec float64          `json:"completedPerSecond"`
	Verification    Verification     `json:"verification"`
	Error           string           `json:"error,omitempty"`
}

// Snapshot is a run as it stands.
type Snapshot struct {
	ID        string           `json:"id"`
	Scenario  Scenario         `json:"scenario"`
	Rate      int              `json:"rate"`
	Duration  float64          `json:"durationSeconds"`
	Wallets   int              `json:"wallets"`
	Phase     Phase            `json:"phase"`
	Elapsed   float64          `json:"elapsedSeconds"`
	Offered   int64            `json:"offered"`
	Sent      int64            `json:"sent"`
	Completed int64            `json:"completed"`
	Dropped   int64            `json:"dropped"`
	Errors    int64            `json:"errors"`
	Replays   int64            `json:"replays"`
	InFlight  int64            `json:"inFlight"`
	Statuses  map[string]int64 `json:"statuses"`
	LatencyMs Latency          `json:"latencyMs"`
	Series    []Point          `json:"series"`
	Result    *Result          `json:"result,omitempty"`
}

type seeded struct {
	id       string
	playerID string
	expected atomic.Int64 // cents the answers say the balance should be
}

// Run is one load run: seed wallets, offer load, drain, verify.
type Run struct {
	ID     string
	spec   Spec
	limits Limits
	target Target
	stats  *stats

	stopLoad context.CancelFunc
	loadCtx  context.Context
	done     chan struct{}

	mu        sync.Mutex
	phase     Phase
	startedAt time.Time
	loadStart time.Time
	loadEnd   time.Time
	wallets   []*seeded
	result    *Result
	unknown   []Operation // sent, never answered
	completed []Operation // processed once, candidates for replay

	// requests tracks the submissions still out when the load ends; only
	// the run's own goroutine sets and reads it.
	requests *sync.WaitGroup
}

// Start validates spec and runs it in the background. ctx bounds the
// whole run, verification included; Stop ends only the load.
func Start(ctx context.Context, target Target, spec Spec, limits Limits) (*Run, error) {
	if err := spec.Validate(limits); err != nil {
		return nil, err
	}
	if limits.RequestTimeout <= 0 {
		limits.RequestTimeout = 10 * time.Second
	}
	if limits.MaxInFlight <= 0 {
		limits.MaxInFlight = 4096
	}
	loadCtx, stopLoad := context.WithCancel(ctx)
	r := &Run{
		ID:        uuid.NewString()[:8],
		spec:      spec,
		limits:    limits,
		target:    target,
		stats:     newStats(),
		loadCtx:   loadCtx,
		stopLoad:  stopLoad,
		done:      make(chan struct{}),
		phase:     PhaseSeeding,
		startedAt: time.Now(),
	}
	go r.execute(ctx)
	return r, nil
}

func (r *Run) Done() <-chan struct{} { return r.done }

// Stop ends the load early; the run still drains and verifies.
func (r *Run) Stop() { r.stopLoad() }

func (r *Run) Result() (Result, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.result == nil {
		return Result{}, false
	}
	return *r.result, true
}

func (r *Run) Snapshot() Snapshot {
	t := r.stats.totals()
	r.mu.Lock()
	defer r.mu.Unlock()
	s := Snapshot{
		ID:        r.ID,
		Scenario:  r.spec.Scenario,
		Rate:      r.spec.Rate,
		Duration:  r.spec.Duration.Seconds(),
		Wallets:   r.spec.walletsToSeed(),
		Phase:     r.phase,
		Elapsed:   time.Since(r.startedAt).Seconds(),
		Offered:   t.Offered,
		Sent:      t.Sent,
		Completed: t.Completed,
		Dropped:   t.Dropped,
		Errors:    t.Errors,
		Replays:   t.Replays,
		InFlight:  t.InFlight,
		Statuses:  t.Statuses,
		LatencyMs: t.Latency,
		Series:    t.Series,
	}
	if r.result != nil {
		result := *r.result
		s.Result = &result
	}
	return s
}

func (r *Run) setPhase(p Phase) {
	r.mu.Lock()
	r.phase = p
	r.mu.Unlock()
}

func (r *Run) execute(ctx context.Context) {
	defer close(r.done)
	defer r.stopLoad()

	if err := r.seed(ctx); err != nil {
		r.finish(PhaseFailed, Verification{}, fmt.Errorf("seeding wallets: %w", err))
		return
	}

	r.setPhase(PhaseRunning)
	r.offerLoad()
	r.setPhase(PhaseDraining)
	r.drain()

	r.setPhase(PhaseVerifying)
	verification := r.verify(ctx)
	final := PhaseDone
	if r.loadEnd.Sub(r.loadStart) < r.spec.Duration-tick {
		final = PhaseStopped
	}
	if ctx.Err() != nil {
		final = PhaseFailed
	}
	r.finish(final, verification, ctx.Err())
}

func (r *Run) finish(phase Phase, v Verification, err error) {
	t := r.stats.totals()
	r.mu.Lock()
	defer r.mu.Unlock()
	loadSeconds := r.loadEnd.Sub(r.loadStart).Seconds()
	result := &Result{
		Phase:        phase,
		Offered:      t.Offered,
		Sent:         t.Sent,
		Completed:    t.Completed,
		Dropped:      t.Dropped,
		Errors:       t.Errors,
		Replays:      t.Replays,
		Statuses:     t.Statuses,
		LatencyMs:    t.Latency,
		LoadSeconds:  loadSeconds,
		Verification: v,
	}
	if loadSeconds > 0 {
		result.CompletedPerSec = float64(t.Completed) / loadSeconds
	}
	if err != nil {
		result.Error = err.Error()
		result.Verification.Passed = false
	}
	r.result = result
	r.phase = phase
}

// seed opens the scenario's wallets, setupWorkers at a time.
func (r *Run) seed(ctx context.Context) error {
	wallets := make([]*seeded, r.spec.walletsToSeed())
	err := forEach(ctx, len(wallets), func(ctx context.Context, i int) error {
		playerID := uuid.NewString()
		id, err := r.target.OpenWallet(ctx, playerID, initialBalance)
		if err != nil {
			return err
		}
		w := &seeded{id: id, playerID: playerID}
		w.expected.Store(int64(initialBalance))
		wallets[i] = w
		return nil
	})
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.wallets = wallets
	r.mu.Unlock()
	return nil
}

// offerLoad issues arrivals at the asked rate for the asked duration,
// whatever the answers do: an open model. An arrival finding MaxInFlight
// requests already waiting is dropped and counted, not delayed.
func (r *Run) offerLoad() {
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	var wg sync.WaitGroup
	defer func() { r.requests = &wg }()

	start := time.Now()
	r.mu.Lock()
	r.loadStart = start
	r.mu.Unlock()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	second := time.NewTicker(time.Second)
	defer second.Stop()

	var issued, seq int64
	issueUntil := func(elapsed time.Duration) {
		due := int64(elapsed.Seconds() * float64(r.spec.Rate))
		for ; issued < due; issued++ {
			r.stats.offered.Add(1)
			if r.stats.inFlight.Load() >= int64(r.limits.MaxInFlight) {
				r.stats.dropped.Add(1)
				continue
			}
			op := r.next(rng, &seq)
			r.stats.sent.Add(1)
			r.stats.inFlight.Add(1)
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer r.stats.inFlight.Add(-1)
				r.send(op)
			}()
		}
	}

	seconds := 0
	for {
		select {
		case <-r.loadCtx.Done():
			r.endLoad(time.Now(), seconds)
			return
		case <-second.C:
			seconds++
			r.stats.point(seconds)
		case now := <-ticker.C:
			elapsed := now.Sub(start)
			if elapsed >= r.spec.Duration {
				issueUntil(r.spec.Duration)
				r.endLoad(start.Add(r.spec.Duration), seconds)
				return
			}
			issueUntil(elapsed)
		}
	}
}

func (r *Run) endLoad(at time.Time, seconds int) {
	r.mu.Lock()
	r.loadEnd = at
	r.mu.Unlock()
	r.stats.point(seconds + 1)
}

// next builds the next operation of the scenario.
func (r *Run) next(rng *rand.Rand, seq *int64) Operation {
	if r.spec.Scenario == ScenarioReplay && rng.Float64() < replayShare {
		r.mu.Lock()
		var op Operation
		ok := len(r.completed) > 0
		if ok {
			op = r.completed[rng.IntN(len(r.completed))]
		}
		r.mu.Unlock()
		if ok {
			op.Replay = true
			return op
		}
	}

	*seq++
	i := 0
	if len(r.wallets) > 1 {
		i = rng.IntN(len(r.wallets))
	}
	kind, amount := KindBet, betAmount
	if rng.Float64() < winShare {
		kind, amount = KindWin, winAmount
	}
	externalID := fmt.Sprintf("loadlab-%s-%d", r.ID, *seq)
	return Operation{
		ExternalID:     externalID,
		IdempotencyKey: externalID,
		WalletID:       r.wallets[i].id,
		PlayerID:       r.wallets[i].playerID,
		Kind:           kind,
		Amount:         amount,
		wallet:         i,
	}
}

// send submits op and books its answer. Requests are bounded by the run's
// context, not the load's, so stopping the load lets them finish.
func (r *Run) send(op Operation) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.loadCtx), r.limits.RequestTimeout)
	defer cancel()

	started := time.Now()
	outcome, err := r.target.Submit(ctx, op)
	if err != nil {
		r.stats.unanswered()
		if !op.Replay {
			r.mu.Lock()
			r.unknown = append(r.unknown, op)
			r.mu.Unlock()
		}
		return
	}
	r.stats.answered(outcome.HTTPStatus, time.Since(started))
	if outcome.Replay {
		r.stats.replays.Add(1)
		return
	}
	if outcome.Status == "PROCESSED" && !op.Replay {
		r.book(op)
	}
}

// book applies a first-time processed operation to the books and offers
// it for replay.
func (r *Run) book(op Operation) {
	r.wallets[op.wallet].expected.Add(int64(op.Delta()))
	if r.spec.Scenario != ScenarioReplay {
		return
	}
	r.mu.Lock()
	if len(r.completed) < replayPool {
		r.completed = append(r.completed, op)
	} else {
		r.completed[rand.IntN(replayPool)] = op
	}
	r.mu.Unlock()
}

// drain waits for the submissions still out, at most one request timeout.
func (r *Run) drain() {
	if r.requests == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		r.requests.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(r.limits.RequestTimeout + time.Second):
	}
}

// verify first settles the operations whose answer was lost, by asking
// for them by external ID, then compares every wallet with the books and
// asks the API to reconcile it.
func (r *Run) verify(ctx context.Context) Verification {
	var v Verification
	r.mu.Lock()
	unknown := append([]Operation(nil), r.unknown...)
	wallets := r.wallets
	r.mu.Unlock()

	var resolved, unresolved atomic.Int64
	err := forEach(ctx, len(unknown), func(ctx context.Context, i int) error {
		op := unknown[i]
		status, found, err := r.target.Lookup(ctx, op.ExternalID)
		if err != nil {
			unresolved.Add(1)
			return nil
		}
		resolved.Add(1)
		if found && status == "PROCESSED" {
			r.wallets[op.wallet].expected.Add(int64(op.Delta()))
		}
		return nil
	})
	v.Resolved, v.Unresolved = int(resolved.Load()), int(unresolved.Load())
	if err != nil {
		v.Error = err.Error()
		return v
	}

	var mu sync.Mutex
	var inconsistent atomic.Int64
	err = forEach(ctx, len(wallets), func(ctx context.Context, i int) error {
		w := wallets[i]
		balance, err := r.target.Balance(ctx, w.id)
		if err != nil {
			return fmt.Errorf("balance of %s: %w", w.id, err)
		}
		if expected := Cents(w.expected.Load()); balance != expected {
			mu.Lock()
			v.Mismatches = append(v.Mismatches, Mismatch{WalletID: w.id, Expected: expected, Actual: balance})
			mu.Unlock()
		}
		consistent, err := r.target.Reconcile(ctx, w.id)
		if err != nil {
			return fmt.Errorf("reconciling %s: %w", w.id, err)
		}
		if !consistent {
			inconsistent.Add(1)
		}
		return nil
	})
	v.WalletsChecked = len(wallets)
	v.InconsistentReconciliations = int(inconsistent.Load())
	if err != nil {
		v.Error = err.Error()
	}
	v.Passed = v.Error == "" && len(v.Mismatches) == 0 && v.InconsistentReconciliations == 0 && v.Unresolved == 0
	return v
}

// forEach calls fn for 0..n-1 with setupWorkers goroutines, stopping at
// the first error.
func forEach(ctx context.Context, n int, fn func(ctx context.Context, i int) error) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(setupWorkers, max(n, 1)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				if err := fn(ctx, i); err != nil {
					cancel(err)
				}
			}
		}()
	}
	for i := range n {
		select {
		case next <- i:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(next)
	wg.Wait()
	if err := context.Cause(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return ctx.Err()
}
