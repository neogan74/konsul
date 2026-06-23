package namespace

import (
	"fmt"
	"regexp"
)

var nameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// ValidateName returns ErrInvalidNamespace if name does not match DNS-label rules.
func ValidateName(name string) error {
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("%w: %q (must match ^[a-z0-9][a-z0-9-]{0,62}$)", ErrInvalidNamespace, name)
	}
	return nil
}

// IsProtected reports whether a namespace cannot be deleted.
func IsProtected(name string) bool {
	return name == DefaultNamespace || name == SystemNamespace
}
