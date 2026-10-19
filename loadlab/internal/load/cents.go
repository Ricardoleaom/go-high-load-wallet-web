package load

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Cents is an amount in hundredths, the wallet API's precision. The load
// lab keeps its own books in Cents so they can be compared exactly with
// the balances the API reports.
type Cents int64

func (c Cents) String() string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s%d.%02d", sign, c/100, c%100)
}

var decimal = regexp.MustCompile(`^-?[0-9]+\.[0-9]{2}$`)

// ParseCents reads the API's decimal strings, which always carry two
// fraction digits.
func ParseCents(text string) (Cents, error) {
	if !decimal.MatchString(text) {
		return 0, fmt.Errorf("amount %q is not a decimal with two fraction digits", text)
	}
	value, err := strconv.ParseInt(strings.Replace(text, ".", "", 1), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount %q: %w", text, err)
	}
	return Cents(value), nil
}
