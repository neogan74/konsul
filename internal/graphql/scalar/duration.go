package scalar

import (
	"fmt"
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// MarshalDuration marshals time.Duration to string (e.g., "30s")
func MarshalDuration(d time.Duration) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, fmt.Sprintf(`"%s"`, d.String()))
	})
}

// UnmarshalDuration unmarshals string to time.Duration
func UnmarshalDuration(v interface{}) (time.Duration, error) {
	if tmpStr, ok := v.(string); ok {
		return time.ParseDuration(tmpStr)
	}
	return 0, fmt.Errorf("duration must be a string")
}
