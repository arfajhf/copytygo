package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Errors map[string][]string
type Validator struct {
	data   map[string]string
	errors Errors
}

func New(data map[string]string) *Validator { return &Validator{data: data, errors: make(Errors)} }
func (v *Validator) Required(field string) *Validator {
	if strings.TrimSpace(v.data[field]) == "" {
		v.add(field, "is required")
	}
	return v
}
func (v *Validator) Email(field string) *Validator {
	value := strings.TrimSpace(v.data[field])
	if value != "" {
		ok, _ := regexp.MatchString(`^[^\s@]+@[^\s@]+\.[^\s@]+$`, value)
		if !ok {
			v.add(field, "must be a valid email")
		}
	}
	return v
}
func (v *Validator) Min(field string, n int) *Validator {
	if len(v.data[field]) < n {
		v.add(field, fmt.Sprintf("must contain at least %d characters", n))
	}
	return v
}
func (v *Validator) Max(field string, n int) *Validator {
	if len(v.data[field]) > n {
		v.add(field, fmt.Sprintf("must contain at most %d characters", n))
	}
	return v
}
func (v *Validator) Integer(field string) *Validator {
	if v.data[field] != "" {
		if _, err := strconv.ParseInt(v.data[field], 10, 64); err != nil {
			v.add(field, "must be an integer")
		}
	}
	return v
}
func (v *Validator) OneOf(field string, values ...string) *Validator {
	value := v.data[field]
	for _, x := range values {
		if value == x {
			return v
		}
	}
	v.add(field, "contains an unsupported value")
	return v
}
func (v *Validator) Valid() bool               { return len(v.errors) == 0 }
func (v *Validator) Errors() Errors            { return v.errors }
func (v *Validator) add(field, message string) { v.errors[field] = append(v.errors[field], message) }
