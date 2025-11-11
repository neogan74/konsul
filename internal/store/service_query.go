package store

import (
	"sort"
	"time"
)

// QueryByTags returns services that have ALL specified tags (AND logic)
func (s *ServiceStore) QueryByTags(tags []string) []Service {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	// If no tags specified, return all services
	if len(tags) == 0 {
		return s.List()
	}

	// Get services with first tag
	var candidateServices map[string]bool
	if services, ok := s.TagIndex[tags[0]]; ok {
		candidateServices = make(map[string]bool, len(services))
		for name := range services {
			candidateServices[name] = true
		}
	} else {
		return []Service{} // No services with first tag
	}

	// Intersect with other tags (AND logic)
	for _, tag := range tags[1:] {
		if services, ok := s.TagIndex[tag]; ok {
			for name := range candidateServices {
				if !services[name] {
					delete(candidateServices, name)
				}
			}
		} else {
			return []Service{} // No services with this tag
		}
	}

	// Build result list (filter expired) with deterministic ordering
	names := make([]string, 0, len(candidateServices))
	for name := range candidateServices {
		names = append(names, name)
	}
	sort.Strings(names)

	now := time.Now()
	result := make([]Service, 0, len(names))
	for _, name := range names {
		if entry, ok := s.Data[name]; ok && entry.ExpiresAt.After(now) {
			result = append(result, entry.Service)
		}
	}

	return result
}

// QueryByMetadata returns services that match ALL specified metadata filters (AND logic)
func (s *ServiceStore) QueryByMetadata(filters map[string]string) []Service {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	// If no filters specified, return all services
	if len(filters) == 0 {
		return s.List()
	}

	// Get candidate services from first filter
	var candidateServices map[string]bool
	var firstKey string
	var firstValue string
	for k, v := range filters {
		firstKey = k
		firstValue = v
		break
	}

	if values, ok := s.MetaIndex[firstKey]; ok {
		if services, ok := values[firstValue]; ok {
			candidateServices = make(map[string]bool, len(services))
			for _, name := range services {
				candidateServices[name] = true
			}
		} else {
			return []Service{} // No services with this metadata value
		}
	} else {
		return []Service{} // No services with this metadata key
	}

	// Intersect with other filters (AND logic)
	for key, value := range filters {
		if key == firstKey {
			continue
		}

		if values, ok := s.MetaIndex[key]; ok {
			if services, ok := values[value]; ok {
				serviceSet := make(map[string]bool)
				for _, name := range services {
					serviceSet[name] = true
				}

				for name := range candidateServices {
					if !serviceSet[name] {
						delete(candidateServices, name)
					}
				}
			} else {
				return []Service{} // No services with this metadata value
			}
		} else {
			return []Service{} // No services with this metadata key
		}
	}

	// Build result list (filter expired) with deterministic ordering
	names := make([]string, 0, len(candidateServices))
	for name := range candidateServices {
		names = append(names, name)
	}
	sort.Strings(names)

	now := time.Now()
	result := make([]Service, 0, len(names))
	for _, name := range names {
		if entry, ok := s.Data[name]; ok && entry.ExpiresAt.After(now) {
			result = append(result, entry.Service)
		}
	}

	return result
}

// QueryByTagsAndMetadata returns services that match both tag and metadata filters (AND logic across all filters)
func (s *ServiceStore) QueryByTagsAndMetadata(tags []string, meta map[string]string) []Service {
	// If no filters, return all services
	if len(tags) == 0 && len(meta) == 0 {
		return s.List()
	}

	// If only tags, use tag query
	if len(tags) > 0 && len(meta) == 0 {
		return s.QueryByTags(tags)
	}

	// If only metadata, use metadata query
	if len(tags) == 0 && len(meta) > 0 {
		return s.QueryByMetadata(meta)
	}

	// Both tags and metadata - intersect results
	tagServices := s.QueryByTags(tags)
	if len(tagServices) == 0 {
		return []Service{}
	}

	metaServices := s.QueryByMetadata(meta)
	if len(metaServices) == 0 {
		return []Service{}
	}

	// Create a set of service names from tag results
	tagServiceSet := make(map[string]bool)
	for _, svc := range tagServices {
		tagServiceSet[svc.Name] = true
	}

	// Filter metadata results to only include those also in tag results
	result := make([]Service, 0)
	for _, svc := range metaServices {
		if tagServiceSet[svc.Name] {
			result = append(result, svc)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}
