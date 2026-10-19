package load

import (
	"errors"
	"fmt"
	"time"
)

type Scenario string

const (
	// ScenarioUniform spreads bets and wins over every seeded wallet.
	ScenarioUniform Scenario = "uniform"
	// ScenarioHotWallet sends everything to one wallet, which serializes
	// on its row lock.
	ScenarioHotWallet Scenario = "hot-wallet"
	// ScenarioReplay resends half of the time an operation that already
	// completed, which the API must answer as a replay without moving
	// money again.
	ScenarioReplay Scenario = "replay"
)

// Spec is what an operator asks for.
type Spec struct {
	Scenario Scenario
	Rate     int // arrivals per second
	Duration time.Duration
	Wallets  int
}

// Limits bound what a run may ask of this machine and of the API.
type Limits struct {
	MaxRate        int
	MaxDuration    time.Duration
	MaxWallets     int
	MaxInFlight    int           // past it, arrivals are dropped and counted
	RequestTimeout time.Duration // zero means 10 s
}

const minDuration = time.Second

func (s Spec) Validate(l Limits) error {
	var errs []error
	switch s.Scenario {
	case ScenarioUniform, ScenarioHotWallet, ScenarioReplay:
	default:
		errs = append(errs, fmt.Errorf("scenario %q is not one of uniform, hot-wallet, replay", s.Scenario))
	}
	if s.Rate < 1 || s.Rate > l.MaxRate {
		errs = append(errs, fmt.Errorf("rate must be between 1 and %d a second", l.MaxRate))
	}
	if s.Duration < minDuration || s.Duration > l.MaxDuration {
		errs = append(errs, fmt.Errorf("duration must be between %s and %s", minDuration, l.MaxDuration))
	}
	if s.Wallets < 1 || s.Wallets > l.MaxWallets {
		errs = append(errs, fmt.Errorf("wallets must be between 1 and %d", l.MaxWallets))
	}
	return errors.Join(errs...)
}

// walletsToSeed is how many wallets the scenario uses: one for a hot
// wallet, whatever was asked otherwise.
func (s Spec) walletsToSeed() int {
	if s.Scenario == ScenarioHotWallet {
		return 1
	}
	return s.Wallets
}
