package load

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// fakeWallet behaves like the wallet API for what the load lab relies on:
// one operation per external ID, replays answered as such, balances that
// move only on a first PROCESSED operation.
type fakeWallet struct {
	mu        sync.Mutex
	balances  map[string]Cents
	seen      map[string]string // external ID → status
	opened    int
	delay     time.Duration
	block     chan struct{}    // when set, Submit waits for it
	loseEvery int              // every nth new operation is applied but its answer lost
	skew      map[string]Cents // wallet → amount Balance adds to the truth
	submitted int
}

func newFakeWallet() *fakeWallet {
	return &fakeWallet{
		balances: map[string]Cents{},
		seen:     map[string]string{},
		skew:     map[string]Cents{},
	}
}

func (f *fakeWallet) OpenWallet(_ context.Context, _ string, initial Cents) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opened++
	id := fmt.Sprintf("wallet-%d", f.opened)
	f.balances[id] = initial
	return id, nil
}

func (f *fakeWallet) Submit(ctx context.Context, op Operation) (Outcome, error) {
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return Outcome{}, ctx.Err()
		}
	}
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return Outcome{}, ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.submitted++
	if status, ok := f.seen[op.ExternalID]; ok {
		return Outcome{HTTPStatus: 200, Status: status, Replay: true}, nil
	}
	f.seen[op.ExternalID] = "PROCESSED"
	f.balances[op.WalletID] += op.Delta()
	if f.loseEvery > 0 && len(f.seen)%f.loseEvery == 0 {
		// Applied, but the answer never reaches the client.
		return Outcome{}, errors.New("connection reset")
	}
	return Outcome{HTTPStatus: 200, Status: "PROCESSED"}, nil
}

func (f *fakeWallet) Lookup(_ context.Context, externalID string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	status, ok := f.seen[externalID]
	return status, ok, nil
}

func (f *fakeWallet) Balance(_ context.Context, walletID string) (Cents, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.balances[walletID] + f.skew[walletID], nil
}

func (f *fakeWallet) Reconcile(_ context.Context, _ string) (bool, error) {
	return true, nil
}
