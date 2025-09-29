package dns

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

func TestDNSServer_StartStop(t *testing.T) {
	serviceStore := store.NewServiceStore()
	log := logger.NewFromConfig("error", "text") // Use error level to reduce test noise

	config := Config{
		Host:   "127.0.0.1",
		Port:   0, // Use any available port
		Domain: "consul",
	}

	server := NewServer(config, serviceStore, log)

	// Start server
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start DNS server: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop DNS server: %v", err)
	}
}

func TestDNSServer_RealQuery(t *testing.T) {
	serviceStore := store.NewServiceStore()
	log := logger.NewFromConfig("error", "text")

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	listener.Close()

	config := Config{
		Host:   "127.0.0.1",
		Port:   port,
		Domain: "consul",
	}

	server := NewServer(config, serviceStore, log)

	// Register test service
	service := store.Service{Name: "test", Address: "10.0.0.1", Port: 8080}
	serviceStore.Register(service)

	// Start server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start DNS server: %v", err)
	}
	defer server.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create DNS client
	client := new(dns.Client)
	client.Timeout = 5 * time.Second

	// Test SRV query
	query := new(dns.Msg)
	query.SetQuestion("_test._tcp.service.consul.", dns.TypeSRV)

	response, _, err := client.Exchange(query, net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)))
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(response.Answer) != 1 {
		t.Errorf("Expected 1 SRV record, got %d", len(response.Answer))
	}

	srv, ok := response.Answer[0].(*dns.SRV)
	if !ok {
		t.Fatal("Expected SRV record")
	}

	if srv.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", srv.Port)
	}

	if srv.Target != "test.node.consul." {
		t.Errorf("Expected target 'test.node.consul.', got '%s'", srv.Target)
	}

	// Check A record in additional section
	if len(response.Extra) != 1 {
		t.Errorf("Expected 1 A record in additional section, got %d", len(response.Extra))
	}

	a, ok := response.Extra[0].(*dns.A)
	if !ok {
		t.Fatal("Expected A record in additional section")
	}

	expectedIP := net.ParseIP("10.0.0.1")
	if !a.A.Equal(expectedIP) {
		t.Errorf("Expected IP %s, got %s", expectedIP, a.A)
	}
}

func TestDNSServer_ServiceLifecycle(t *testing.T) {
	serviceStore := store.NewServiceStore()
	log := logger.NewFromConfig("error", "text")

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	listener.Close()

	config := Config{
		Host:   "127.0.0.1",
		Port:   port,
		Domain: "consul",
	}

	server := NewServer(config, serviceStore, log)

	// Start server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start DNS server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	client := new(dns.Client)
	client.Timeout = 5 * time.Second

	// Test 1: No services registered - should get NXDOMAIN
	query := new(dns.Msg)
	query.SetQuestion("_web._tcp.service.consul.", dns.TypeSRV)

	response, _, err := client.Exchange(query, net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)))
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if response.Rcode != dns.RcodeNameError {
		t.Errorf("Expected NXDOMAIN for non-existent service, got rcode %d", response.Rcode)
	}

	// Test 2: Register service - should get records
	service := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	serviceStore.Register(service)

	response, _, err = client.Exchange(query, net.JoinHostPort("127.0.0.1", string(rune(port))))
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(response.Answer) != 1 {
		t.Errorf("Expected 1 SRV record after registration, got %d", len(response.Answer))
	}

	// Test 3: Deregister service - should get NXDOMAIN again
	serviceStore.Deregister("web")

	response, _, err = client.Exchange(query, net.JoinHostPort("127.0.0.1", string(rune(port))))
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if response.Rcode != dns.RcodeNameError {
		t.Errorf("Expected NXDOMAIN after deregistration, got rcode %d", response.Rcode)
	}
}

func TestDNSServer_LoadBalancing(t *testing.T) {
	serviceStore := store.NewServiceStore()
	log := logger.NewFromConfig("error", "text")

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	listener.Close()

	config := Config{
		Host:   "127.0.0.1",
		Port:   port,
		Domain: "consul",
	}

	server := NewServer(config, serviceStore, log)

	// Register multiple instances of same service
	service1 := store.Service{Name: "api", Address: "10.0.0.1", Port: 8080}
	service2 := store.Service{Name: "api", Address: "10.0.0.2", Port: 8080}
	service3 := store.Service{Name: "api", Address: "10.0.0.3", Port: 8080}

	serviceStore.Register(service1)
	serviceStore.Register(service2)
	serviceStore.Register(service3)

	// Start server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start DNS server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	client := new(dns.Client)
	client.Timeout = 5 * time.Second

	// Query for SRV records
	query := new(dns.Msg)
	query.SetQuestion("_api._tcp.service.consul.", dns.TypeSRV)

	response, _, err := client.Exchange(query, net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)))
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	// Should get 3 SRV records (one for each instance)
	if len(response.Answer) != 3 {
		t.Errorf("Expected 3 SRV records for load balancing, got %d", len(response.Answer))
	}

	// Should get 3 A records in additional section
	if len(response.Extra) != 3 {
		t.Errorf("Expected 3 A records in additional section, got %d", len(response.Extra))
	}

	// Verify different weights for load balancing
	weights := make(map[uint16]bool)
	for _, answer := range response.Answer {
		if srv, ok := answer.(*dns.SRV); ok {
			weights[srv.Weight] = true
		}
	}

	// Should have at least 2 different weights for load balancing
	if len(weights) < 2 {
		t.Errorf("Expected different weights for load balancing, got weights: %v", weights)
	}
}

func TestDNSServer_ConcurrentQueries(t *testing.T) {
	serviceStore := store.NewServiceStore()
	log := logger.NewFromConfig("error", "text")

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	listener.Close()

	config := Config{
		Host:   "127.0.0.1",
		Port:   port,
		Domain: "consul",
	}

	server := NewServer(config, serviceStore, log)

	// Register test service
	service := store.Service{Name: "concurrent", Address: "10.0.0.1", Port: 8080}
	serviceStore.Register(service)

	// Start server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start DNS server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Run concurrent queries
	const numQueries = 10
	results := make(chan error, numQueries)

	for i := 0; i < numQueries; i++ {
		go func() {
			client := new(dns.Client)
			client.Timeout = 5 * time.Second

			query := new(dns.Msg)
			query.SetQuestion("_concurrent._tcp.service.consul.", dns.TypeSRV)

			response, _, err := client.Exchange(query, net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)))
			if err != nil {
				results <- err
				return
			}

			if len(response.Answer) != 1 {
				results <- err
				return
			}

			results <- nil
		}()
	}

	// Wait for all queries to complete
	for i := 0; i < numQueries; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent query %d failed: %v", i, err)
		}
	}
}