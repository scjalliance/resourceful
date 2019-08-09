package enforcer

import (
	"regexp"
	"strings"
)

func compileRegex(re string) (*regexp.Regexp, error) {
	if re == "" {
		return nil, nil
	}

	// Force case-insensitive matching
	if !strings.HasPrefix(re, "(?i)") {
		re = "(?i)" + re
	}

	return regexp.Compile(re)
}
