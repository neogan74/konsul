package store

// addToTagIndex adds a service to the tag index for all its tags
func (s *ServiceStore) addToTagIndex(serviceName string, tags []string) {
	for _, tag := range tags {
		if s.TagIndex[tag] == nil {
			s.TagIndex[tag] = make(map[string]bool)
		}
		s.TagIndex[tag][serviceName] = true
	}
}

// removeFromTagIndex removes a service from the tag index for all its tags
func (s *ServiceStore) removeFromTagIndex(serviceName string, tags []string) {
	for _, tag := range tags {
		if services, ok := s.TagIndex[tag]; ok {
			delete(services, serviceName)
			// Clean up empty tag entries
			if len(services) == 0 {
				delete(s.TagIndex, tag)
			}
		}
	}
}

// addToMetaIndex adds a service to the metadata index for all its metadata
func (s *ServiceStore) addToMetaIndex(serviceName string, meta map[string]string) {
	for key, value := range meta {
		if s.MetaIndex[key] == nil {
			s.MetaIndex[key] = make(map[string][]string)
		}
		s.MetaIndex[key][value] = append(s.MetaIndex[key][value], serviceName)
	}
}

// removeFromMetaIndex removes a service from the metadata index for all its metadata
func (s *ServiceStore) removeFromMetaIndex(serviceName string, meta map[string]string) {
	for key, value := range meta {
		if values, ok := s.MetaIndex[key]; ok {
			if services, ok := values[value]; ok {
				// Remove serviceName from slice
				for i, name := range services {
					if name == serviceName {
						s.MetaIndex[key][value] = append(services[:i], services[i+1:]...)
						break
					}
				}
				// Cleanup empty entries
				if len(s.MetaIndex[key][value]) == 0 {
					delete(s.MetaIndex[key], value)
				}
			}
			if len(s.MetaIndex[key]) == 0 {
				delete(s.MetaIndex, key)
			}
		}
	}
}
