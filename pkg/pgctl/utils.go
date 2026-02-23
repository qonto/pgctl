package pgctl

import (
	"strings"
)

func getStringSeparatedByCommas(array []string) string {
	return strings.Join(array, `,`)
}
