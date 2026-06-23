package namespace

// NamespaceStore manages namespace lifecycle.
type NamespaceStore interface {
	// Create creates a new namespace. Returns ErrNamespaceExists if already present.
	Create(ns *Namespace) error
	// Get returns the namespace by name. Returns ErrNamespaceNotFound if absent.
	Get(name string) (*Namespace, error)
	// List returns all namespaces.
	List() ([]*Namespace, error)
	// Delete removes a namespace. Returns ErrNamespaceReserved for "default".
	Delete(name string) error
	// Exists reports whether the namespace exists.
	Exists(name string) (bool, error)
}
