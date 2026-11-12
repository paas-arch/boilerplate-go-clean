package validation

import (
	"errors"
	"fmt"
	"regexp"
)

func NewError(msg string) error { return errors.New(msg) }

func InEnumString(val string, allowed []string) error {
	for _, v := range allowed {
		if v == val {
			return nil
		}
	}
	return fmt.Errorf("value %q not in allowed values %v", val, allowed)
}

func MaxLen(l, max int) error {
	if l > max {
		return fmt.Errorf("length %d exceeds maximum %d", l, max)
	}
	return nil
}

func CreateMatcher(pattern string) func(string) error {
	re := regexp.MustCompile(pattern)
	return func(s string) error {
		if !re.MatchString(s) {
			return fmt.Errorf("value %q does not match pattern %q", s, pattern)
		}
		return nil
	}
}
