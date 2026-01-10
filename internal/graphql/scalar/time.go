package scalar

import (
	"fmt"
	"io"
	"time"
)

// Time is a custom scalar type for timestamps
type Time time.Time

// MarshalGQL implements the graphql.Marshaler interface
func (t Time) MarshalGQL(w io.Writer) {
	timestamp := time.Time(t).Format(time.RFC3339)
	_, _ = io.WriteString(w, fmt.Sprintf(`"%s"`, timestamp))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (t *Time) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("time must be a string")
	}

	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("failed to parse time: %w", err)
	}

	*t = Time(parsed)
	return nil
}

// ToTime converts scalar.Time to time.Time
func (t Time) ToTime() time.Time {
	return time.Time(t)
}

// FromTime converts time.Time to scalar.Time
func FromTime(t time.Time) Time {
	return Time(t)
}
