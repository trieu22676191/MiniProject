package money

import (
	"fmt"
	"strconv"
	"strings"
)

const MaxDatabaseCents int64 = 999_999_999_999_999_999

func ParseCents(amount string) (int64, error) {
	whole, fraction, ok := strings.Cut(amount, ".")
	if !ok || len(fraction) != 2 || strings.HasPrefix(whole, "-") {
		return 0, fmt.Errorf("invalid money amount %q", amount)
	}
	cents, err := strconv.ParseInt(whole+fraction, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse money amount: %w", err)
	}
	return cents, nil
}

func FormatCents(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
