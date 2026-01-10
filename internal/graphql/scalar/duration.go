package scalar

import (
	"fmt"
	"io"
	"time"
)

// Duration is a custom scalar type for durations
type Duration time.Duration

// MarshalGQL implements the graphql.Marshaler interface
func (d Duration) MarshalGQL(w io.Writer) {
	duration := time.Duration(d).String()
	_, _ = io.WriteString(w, fmt.Sprintf(`"%s"`, duration))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (d *Duration) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("duration must be a string")
	}

	parsed, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("failed to parse duration: %w", err)
	}

	*d = Duration(parsed)
	return nil
}

// ToDuration converts scalar.Duration to time.Duration
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}

// FromDuration converts time.Duration to scalar.Duration
func FromDuration(d time.Duration) Duration {
	return Duration(d)
}
