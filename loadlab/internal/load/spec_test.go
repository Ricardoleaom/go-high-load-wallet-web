package load

import (
	"testing"
	"time"
)

func TestSpecRejectsOutOfLimits(t *testing.T) {
	t.Parallel()
	limits := Limits{MaxRate: 5000, MaxDuration: 10 * time.Minute, MaxWallets: 10000}
	valid := Spec{Scenario: ScenarioUniform, Rate: 1000, Duration: time.Minute, Wallets: 100}
	if err := valid.Validate(limits); err != nil {
		t.Fatalf("valid spec rejected: %v", err)
	}

	for name, spec := range map[string]Spec{
		"unknown scenario": {Scenario: "jackpot", Rate: 10, Duration: time.Second, Wallets: 1},
		"zero rate":        {Scenario: ScenarioUniform, Rate: 0, Duration: time.Second, Wallets: 1},
		"rate over limit":  {Scenario: ScenarioUniform, Rate: 5001, Duration: time.Second, Wallets: 1},
		"too short":        {Scenario: ScenarioUniform, Rate: 10, Duration: 500 * time.Millisecond, Wallets: 1},
		"too long":         {Scenario: ScenarioUniform, Rate: 10, Duration: 11 * time.Minute, Wallets: 1},
		"no wallets":       {Scenario: ScenarioUniform, Rate: 10, Duration: time.Second, Wallets: 0},
		"too many wallets": {Scenario: ScenarioUniform, Rate: 10, Duration: time.Second, Wallets: 10001},
	} {
		if err := spec.Validate(limits); err == nil {
			t.Errorf("%s: accepted %+v", name, spec)
		}
	}
}

func TestCentsReadAndWriteTwoDecimals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		cents Cents
		text  string
	}{{0, "0.00"}, {5, "0.05"}, {100, "1.00"}, {123456, "1234.56"}, {-150, "-1.50"}} {
		if got := c.cents.String(); got != c.text {
			t.Errorf("Cents(%d).String() = %q, want %q", c.cents, got, c.text)
		}
		parsed, err := ParseCents(c.text)
		if err != nil || parsed != c.cents {
			t.Errorf("ParseCents(%q) = %d, %v; want %d", c.text, parsed, err, c.cents)
		}
	}
	for _, bad := range []string{"", "1.5", "1.234", "abc", "1,00"} {
		if _, err := ParseCents(bad); err == nil {
			t.Errorf("ParseCents(%q) accepted", bad)
		}
	}
}
