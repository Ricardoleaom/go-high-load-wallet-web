package load

import "context"

// Operation is one wager the load lab sends.
type Operation struct {
	ExternalID     string
	IdempotencyKey string
	WalletID       string
	PlayerID       string
	Kind           string // BET or WIN
	Amount         Cents
	// Replay marks a resend of an operation that already completed.
	Replay bool

	wallet int // index into the run's wallets
}

// Delta is what the operation does to its wallet's balance when processed.
func (o Operation) Delta() Cents {
	if o.Kind == KindWin {
		return o.Amount
	}
	return -o.Amount
}

const (
	KindBet = "BET"
	KindWin = "WIN"
)

// Outcome is the API's answer to a submission.
type Outcome struct {
	HTTPStatus  int
	Status      string // PROCESSED, REJECTED, FAILED, PENDING_REFERENCE, or empty for an error body
	FailureCode string
	Replay      bool
}

// Target is the wallet API as the load lab uses it. Submit returns an
// error only when no answer came back: the operation may or may not have
// been applied.
type Target interface {
	OpenWallet(ctx context.Context, playerID string, initial Cents) (walletID string, err error)
	Submit(ctx context.Context, op Operation) (Outcome, error)
	Lookup(ctx context.Context, externalID string) (status string, found bool, err error)
	Balance(ctx context.Context, walletID string) (Cents, error)
	Reconcile(ctx context.Context, walletID string) (consistent bool, err error)
}
