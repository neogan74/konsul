package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/template"
)

// KonsulClient is a simple HTTP client for Konsul
type KonsulClient struct {
	addr       string
	httpClient *http.Client
	log        logger.Logger
	kvCache    *KVCache
	svcCache   *ServiceCache
}

// NewKonsulClient creates a new Konsul client
func NewKonsulClient(addr string, log logger.Logger) *KonsulClient {
	return &KonsulClient{
		addr: addr,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		log:      log,
		kvCache:  NewKVCache(),
		svcCache: NewServiceCache(),
	}
}

// KVStore returns the KV store interface
func (c *KonsulClient) KVStore() template.KVStoreReader {
	// Fetch initial data
	_ = c.refreshKV()
	return c.kvCache
}

// ServiceStore returns the service store interface
func (c *KonsulClient) ServiceStore() template.ServiceStoreReader {
	// Fetch initial data
	_ = c.refreshServices()
	return c.svcCache
}

// refreshKV fetches KV data from Konsul
func (c *KonsulClient) refreshKV() error {
	url := fmt.Sprintf("%s/kv", c.addr)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.log.Warn("Failed to fetch KV data", logger.Error(err))
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		c.log.Warn("KV endpoint returned non-200 status",
			logger.Int("status", resp.StatusCode))
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var kvData map[string]string
	if err := json.Unmarshal(body, &kvData); err != nil {
		return err
	}

	c.kvCache.Update(kvData)
	return nil
}

// refreshServices fetches service data from Konsul
func (c *KonsulClient) refreshServices() error {
	url := fmt.Sprintf("%s/services", c.addr)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.log.Warn("Failed to fetch service data", logger.Error(err))
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		c.log.Warn("Services endpoint returned non-200 status",
			logger.Int("status", resp.StatusCode))
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var services []template.Service
	if err := json.Unmarshal(body, &services); err != nil {
		return err
	}

	c.svcCache.Update(services)
	return nil
}

// KVCache caches KV data
type KVCache struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewKVCache creates a new KV cache
func NewKVCache() *KVCache {
	return &KVCache{
		data: make(map[string]string),
	}
}

func (c *KVCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *KVCache) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

func (c *KVCache) Update(data map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
}

// ServiceCache caches service data
type ServiceCache struct {
	mu       sync.RWMutex
	services map[string]template.Service
}

// NewServiceCache creates a new service cache
func NewServiceCache() *ServiceCache {
	return &ServiceCache{
		services: make(map[string]template.Service),
	}
}

func (c *ServiceCache) Get(name string) (template.Service, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	svc, ok := c.services[name]
	return svc, ok
}

func (c *ServiceCache) List() []template.Service {
	c.mu.RLock()
	defer c.mu.RUnlock()
	services := make([]template.Service, 0, len(c.services))
	for _, svc := range c.services {
		services = append(services, svc)
	}
	return services
}

func (c *ServiceCache) Update(services []template.Service) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services = make(map[string]template.Service)
	for _, svc := range services {
		c.services[svc.Name] = svc
	}
}
