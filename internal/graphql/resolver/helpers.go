package resolver

// stringOrEmpty returns empty string if pointer is nil, otherwise returns the value
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
