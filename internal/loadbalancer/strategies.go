package loadbalancer

import (
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/neogan74/konsul/internal/store"
)

// SelectOptions contains parameters for advanced load balancing strategies
type SelectOptions struct {
	ClientIP     string // Client IP for IP hash strategy
	SessionKey   string // Session key for ring hash strategy
	ClientRegion string // Client region for latency-based strategy
}

// selectWeightedRoundRobin implements smooth weighted round-robin selection (NGINX-style)
func (b *Balancer) selectWeightedRoundRobin(serviceName string, instances []store.Service) store.Service {
	if len(instances) == 0 {
		return store.Service{}
	}

	// Extract weights from metadata
	weights := make([]int, len(instances))
	totalWeight := 0
	for i, svc := range instances {
		weight := extractWeight(svc)
		weights[i] = weight
		totalWeight += weight
	}

	// If all weights are 0 or equal, fall back to regular round robin
	if totalWeight == 0 || allEqual(weights) {
		return b.selectRoundRobin(serviceName, instances)
	}

	// Smooth weighted round robin algorithm
	// Track current weights for each instance
	b.mutex.Lock()
	defer b.mutex.Unlock()

	key := "wrr:" + serviceName
	currentWeights, exists := b.getWeightState(key, len(instances))
	if !exists {
		// Initialize current weights to 0
		currentWeights = make([]int, len(instances))
		b.setWeightState(key, currentWeights)
	}

	// Add weights to current weights
	maxIdx := 0
	maxCurrent := math.MinInt32
	for i := range instances {
		currentWeights[i] += weights[i]
		if currentWeights[i] > maxCurrent {
			maxCurrent = currentWeights[i]
			maxIdx = i
		}
	}

	// Subtract total weight from selected instance
	currentWeights[maxIdx] -= totalWeight
	b.setWeightState(key, currentWeights)

	return instances[maxIdx]
}

// selectWeightedRandom implements weighted random selection
func (b *Balancer) selectWeightedRandom(instances []store.Service) store.Service {
	if len(instances) == 0 {
		return store.Service{}
	}

	// Extract weights
	weights := make([]int, len(instances))
	totalWeight := 0
	for i, svc := range instances {
		weight := extractWeight(svc)
		if weight <= 0 {
			weight = 1
		}
		weights[i] = weight
		totalWeight += weight
	}

	// If all weights equal, fall back to regular random
	if allEqual(weights) {
		return b.selectRandom(instances)
	}

	// Select based on weighted probability
	randVal := rand.Intn(totalWeight)
	cumulative := 0
	for i, weight := range weights {
		cumulative += weight
		if randVal < cumulative {
			return instances[i]
		}
	}

	// Fallback (should not reach here)
	return instances[len(instances)-1]
}

// selectIPHash implements IP hash-based sticky session selection
func (b *Balancer) selectIPHash(instances []store.Service, clientIP string) store.Service {
	if len(instances) == 0 || clientIP == "" {
		return b.selectRandom(instances)
	}

	// Hash client IP
	hash := fnv.New64a()
	hash.Write([]byte(clientIP))
	hashValue := hash.Sum64()

	// Select instance using modulo
	idx := int(hashValue % uint64(len(instances)))
	return instances[idx]
}

// selectRingHash implements consistent hashing (ring hash) selection
func (b *Balancer) selectRingHash(serviceName string, instances []store.Service, sessionKey string) store.Service {
	if len(instances) == 0 || sessionKey == "" {
		return b.selectRandom(instances)
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Get or create hash ring for this service
	ring := b.getOrCreateRing(serviceName, instances)

	// Hash the session key
	hash := fnv.New64a()
	hash.Write([]byte(sessionKey))
	hashValue := hash.Sum64()

	// Binary search for closest node
	idx := sort.Search(len(ring.nodes), func(i int) bool {
		return ring.nodes[i] >= hashValue
	})

	if idx == len(ring.nodes) {
		idx = 0 // Wrap around
	}

	instanceIdx := ring.instances[ring.nodes[idx]]
	return instances[instanceIdx]
}

// selectLatencyBased implements latency-based selection using region tags
func (b *Balancer) selectLatencyBased(instances []store.Service, clientRegion string) store.Service {
	if len(instances) == 0 {
		return store.Service{}
	}

	// If no client region specified, fall back to random
	if clientRegion == "" {
		return b.selectRandom(instances)
	}

	// Try to find instance in same region
	for _, svc := range instances {
		region := extractRegionFromTags(svc.Tags)
		if region == clientRegion {
			return svc
		}
	}

	// Fallback: select random instance
	return b.selectRandom(instances)
}

// Helper: extract weight from service metadata
func extractWeight(svc store.Service) int {
	if svc.Meta == nil {
		return 1 // Default weight
	}

	weightStr, ok := svc.Meta["weight"]
	if !ok {
		return 1
	}

	weight, err := strconv.Atoi(weightStr)
	if err != nil || weight <= 0 {
		return 1
	}

	return weight
}

// Helper: check if all values are equal
func allEqual(values []int) bool {
	if len(values) == 0 {
		return true
	}
	first := values[0]
	for _, v := range values[1:] {
		if v != first {
			return false
		}
	}
	return true
}

// Helper: extract region from tags (looks for "region:xxx" tag)
func extractRegionFromTags(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "region:") {
			return strings.TrimPrefix(tag, "region:")
		}
	}
	return ""
}

// hashRing represents a consistent hash ring
type hashRing struct {
	nodes     []uint64       // Sorted hash values
	instances map[uint64]int // Hash -> instance index
	replicas  int            // Virtual nodes per instance
}

// Weight state storage
var (
	weightStates      = make(map[string][]int)
	weightStatesMutex sync.RWMutex
)

func (b *Balancer) getWeightState(key string, size int) ([]int, bool) {
	weightStatesMutex.RLock()
	defer weightStatesMutex.RUnlock()
	state, exists := weightStates[key]
	if exists && len(state) == size {
		return state, true
	}
	return nil, false
}

func (b *Balancer) setWeightState(key string, state []int) {
	weightStatesMutex.Lock()
	defer weightStatesMutex.Unlock()
	weightStates[key] = state
}

// Hash ring storage
var (
	hashRings      = make(map[string]*hashRing)
	hashRingsMutex sync.RWMutex
)

func (b *Balancer) getOrCreateRing(serviceName string, instances []store.Service) *hashRing {
	hashRingsMutex.Lock()
	defer hashRingsMutex.Unlock()

	// Check if ring exists and is valid
	ring, exists := hashRings[serviceName]
	if exists && len(ring.instances) == len(instances) {
		return ring
	}

	// Create new ring
	replicas := 150 // Virtual nodes per instance
	ring = &hashRing{
		nodes:     make([]uint64, 0, len(instances)*replicas),
		instances: make(map[uint64]int),
		replicas:  replicas,
	}

	// Add virtual nodes for each instance
	for i, svc := range instances {
		for r := 0; r < replicas; r++ {
			hash := fnv.New64a()
			key := svc.Name + ":" + svc.Address + ":" + strconv.Itoa(r)
			hash.Write([]byte(key))
			hashValue := hash.Sum64()

			ring.nodes = append(ring.nodes, hashValue)
			ring.instances[hashValue] = i
		}
	}

	// Sort nodes for binary search
	sort.Slice(ring.nodes, func(i, j int) bool {
		return ring.nodes[i] < ring.nodes[j]
	})

	hashRings[serviceName] = ring
	return ring
}
