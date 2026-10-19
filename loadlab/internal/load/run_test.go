package load

import (
	"context"
	"testing"
	"time"
)

func testLimits() Limits {
	return Limits{MaxRate: 5000, MaxDuration: time.Minute, MaxWallets: 1000, MaxInFlight: 4096}
}

// runToEnd starts spec against target and waits for the result.
func runToEnd(t *testing.T, target Target, spec Spec, limits Limits) Result {
	t.Helper()
	run, err := Start(context.Background(), target, spec, limits)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case <-run.Done():
	case <-time.After(spec.Duration + 30*time.Second):
		t.Fatal("run did not finish")
	}
	result, ok := run.Result()
	if !ok {
		t.Fatal("finished run has no result")
	}
	return result
}

func within(got, want int64, fraction float64) bool {
	diff := float64(got - want)
	return diff <= float64(want)*fraction && -diff <= float64(want)*fraction
}

func TestOpenModelKeepsTheRateWhenTheServerIsSlow(t *testing.T) {
	t.Parallel()
	target := newFakeWallet()
	target.delay = 200 * time.Millisecond

	result := runToEnd(t, target, Spec{Scenario: ScenarioUniform, Rate: 500, Duration: 2 * time.Second, Wallets: 10}, testLimits())

	// A closed model with as many workers as the first second needs would
	// send far fewer; arrivals here do not wait for answers.
	if !within(result.Offered, 1000, 0.05) || result.Dropped != 0 {
		t.Fatalf("offered %d, dropped %d; want about 1000 and none", result.Offered, result.Dropped)
	}
	if !within(result.Completed, 1000, 0.05) {
		t.Fatalf("completed %d, want about 1000", result.Completed)
	}
	if result.LatencyMs.P50 < 190 {
		t.Fatalf("p50 %.1f ms, want the server's 200 ms", result.LatencyMs.P50)
	}
}

func TestDropsArrivalsPastMaxInFlight(t *testing.T) {
	t.Parallel()
	target := newFakeWallet()
	target.block = make(chan struct{})
	limits := testLimits()
	limits.MaxInFlight = 10

	run, err := Start(context.Background(), target, Spec{Scenario: ScenarioUniform, Rate: 200, Duration: time.Second, Wallets: 5}, limits)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(1200 * time.Millisecond)
	close(target.block)
	<-run.Done()
	result, _ := run.Result()

	if result.Sent != 10 || !within(result.Dropped, 190, 0.06) {
		t.Fatalf("sent %d, dropped %d; want 10 and about 190", result.Sent, result.Dropped)
	}
	if result.Offered != result.Sent+result.Dropped {
		t.Fatalf("offered %d != sent %d + dropped %d", result.Offered, result.Sent, result.Dropped)
	}
}

func TestAccountsEachOperationOnceUnderReplay(t *testing.T) {
	t.Parallel()
	target := newFakeWallet()

	result := runToEnd(t, target, Spec{Scenario: ScenarioReplay, Rate: 400, Duration: 2 * time.Second, Wallets: 8}, testLimits())

	if result.Replays == 0 {
		t.Fatal("replay scenario sent no replays")
	}
	if !result.Verification.Passed {
		t.Fatalf("verification failed: %+v", result.Verification)
	}
	if result.Verification.WalletsChecked != 8 {
		t.Fatalf("checked %d wallets, want 8", result.Verification.WalletsChecked)
	}
}

func TestHotWalletSendsEverythingToOneWallet(t *testing.T) {
	t.Parallel()
	target := newFakeWallet()

	result := runToEnd(t, target, Spec{Scenario: ScenarioHotWallet, Rate: 200, Duration: time.Second, Wallets: 50}, testLimits())

	if result.Verification.WalletsChecked != 1 || !result.Verification.Passed {
		t.Fatalf("verification %+v, want one wallet checked and passed", result.Verification)
	}
}

func TestVerificationFlagsAWalletWhoseBalanceDiffers(t *testing.T) {
	t.Parallel()
	target := newFakeWallet()
	target.skew["wallet-2"] = 7

	result := runToEnd(t, target, Spec{Scenario: ScenarioUniform, Rate: 100, Duration: time.Second, Wallets: 3}, testLimits())

	v := result.Verification
	if v.Passed || len(v.Mismatches) != 1 || v.Mismatches[0].WalletID != "wallet-2" {
		t.Fatalf("verification %+v, want one mismatch on wallet-2", v)
	}
	if v.Mismatches[0].Actual-v.Mismatches[0].Expected != 7 {
		t.Fatalf("mismatch %+v, want a difference of 7 cents", v.Mismatches[0])
	}
}

func TestUnknownOutcomesAreResolvedBeforeVerifying(t *testing.T) {
	t.Parallel()
	target := newFakeWallet()
	target.loseEvery = 10

	result := runToEnd(t, target, Spec{Scenario: ScenarioUniform, Rate: 200, Duration: time.Second, Wallets: 4}, testLimits())

	if result.Errors == 0 {
		t.Fatal("no answer was lost; the test proves nothing")
	}
	if int64(result.Verification.Resolved) != result.Errors {
		t.Fatalf("resolved %d of %d lost answers", result.Verification.Resolved, result.Errors)
	}
	if !result.Verification.Passed {
		t.Fatalf("verification failed after resolving lost answers: %+v", result.Verification)
	}
}

func TestStopEndsTheRunAndStillVerifies(t *testing.T) {
	t.Parallel()
	target := newFakeWallet()
	run, err := Start(context.Background(), target, Spec{Scenario: ScenarioUniform, Rate: 100, Duration: time.Minute, Wallets: 2}, testLimits())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	run.Stop()
	select {
	case <-run.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("stopped run did not finish")
	}
	result, _ := run.Result()
	if result.Phase != PhaseStopped || !result.Verification.Passed || result.Offered > 100 {
		t.Fatalf("result %+v, want stopped early, verified", result)
	}
}

func TestSnapshotsCarryOnePointPerSecond(t *testing.T) {
	t.Parallel()
	target := newFakeWallet()
	run, err := Start(context.Background(), target, Spec{Scenario: ScenarioUniform, Rate: 300, Duration: 3 * time.Second, Wallets: 3}, testLimits())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-run.Done()

	snapshot := run.Snapshot()
	if n := len(snapshot.Series); n < 2 || n > 4 {
		t.Fatalf("got %d points for a 3-second run", n)
	}
	for _, p := range snapshot.Series[:len(snapshot.Series)-1] {
		if !within(p.Sent, 300, 0.1) {
			t.Fatalf("point %+v, want about 300 sent", p)
		}
	}
}
