package dns

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

func setupTestServer() (*Server, *store.ServiceStore) {
	serviceStore := store.NewServiceStoreWithTTL(30 * time.Second)
	log := logger.NewFromConfig("info", "text")

	config := Config{
		Host:   "localhost",
		Port:   0, // Use any available port for testing
		Domain: "consul",
	}

	dnsServer := NewServer(config, serviceStore, log)
	return dnsServer, serviceStore
}

func TestDNSServer_SRVQuery(t *testing.T) {
	dnsServer, serviceStore := setupTestServer()

	// Register test service (only one instance per name in current implementation)
	service1 := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	serviceStore.Register(service1)

	// Create DNS query
	query := new(dns.Msg)
	query.SetQuestion("_web._tcp.service.consul.", dns.TypeSRV)

	// Create mock response writer
	mockWriter := &mockResponseWriter{}

	// Handle the query
	dnsServer.handleDNSRequest(mockWriter, query)

	// Verify response
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	if len(mockWriter.msg.Answer) != 1 {
		t.Errorf("Expected 1 SRV record, got %d", len(mockWriter.msg.Answer))
	}

	// Check first SRV record
	srv, ok := mockWriter.msg.Answer[0].(*dns.SRV)
	if !ok {
		t.Fatal("Expected SRV record")
	}

	if srv.Priority != 1 {
		t.Errorf("Expected priority 1, got %d", srv.Priority)
	}

	if srv.Port != 80 {
		t.Errorf("Expected port 80, got %d", srv.Port)
	}

	if srv.Target != "web.node.consul." {
		t.Errorf("Expected target 'web.node.consul.', got '%s'", srv.Target)
	}

	// Check that A records are in additional section
	if len(mockWriter.msg.Extra) != 1 {
		t.Errorf("Expected 1 A record in additional section, got %d", len(mockWriter.msg.Extra))
	}

	a, ok := mockWriter.msg.Extra[0].(*dns.A)
	if !ok {
		t.Fatal("Expected A record in additional section")
	}

	expectedIP := net.ParseIP("192.168.1.100")
	if !a.A.Equal(expectedIP) {
		t.Errorf("Expected IP %s, got %s", expectedIP, a.A)
	}
}

func TestDNSServer_AQuery_NodeFormat(t *testing.T) {
	dnsServer, serviceStore := setupTestServer()

	// Register test service
	service := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	serviceStore.Register(service)

	// Create DNS query for node format
	query := new(dns.Msg)
	query.SetQuestion("web.node.consul.", dns.TypeA)

	mockWriter := &mockResponseWriter{}
	dnsServer.handleDNSRequest(mockWriter, query)

	// Verify response
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	if len(mockWriter.msg.Answer) != 1 {
		t.Errorf("Expected 1 A record, got %d", len(mockWriter.msg.Answer))
	}

	a, ok := mockWriter.msg.Answer[0].(*dns.A)
	if !ok {
		t.Fatal("Expected A record")
	}

	expectedIP := net.ParseIP("192.168.1.100")
	if !a.A.Equal(expectedIP) {
		t.Errorf("Expected IP %s, got %s", expectedIP, a.A)
	}
}

func TestDNSServer_AQuery_ServiceFormat(t *testing.T) {
	dnsServer, serviceStore := setupTestServer()

	// Register test service
	service := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	serviceStore.Register(service)

	// Create DNS query for service format
	query := new(dns.Msg)
	query.SetQuestion("web.service.consul.", dns.TypeA)

	mockWriter := &mockResponseWriter{}
	dnsServer.handleDNSRequest(mockWriter, query)

	// Verify response
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	if len(mockWriter.msg.Answer) != 1 {
		t.Errorf("Expected 1 A record, got %d", len(mockWriter.msg.Answer))
	}

	a, ok := mockWriter.msg.Answer[0].(*dns.A)
	if !ok {
		t.Fatal("Expected A record")
	}

	expectedIP := net.ParseIP("192.168.1.100")
	if !a.A.Equal(expectedIP) {
		t.Errorf("Expected IP %s, got %s", expectedIP, a.A)
	}
}

func TestDNSServer_NonExistentService(t *testing.T) {
	dnsServer, _ := setupTestServer()

	// Query for non-existent service
	query := new(dns.Msg)
	query.SetQuestion("_nonexistent._tcp.service.consul.", dns.TypeSRV)

	mockWriter := &mockResponseWriter{}
	dnsServer.handleDNSRequest(mockWriter, query)

	// Verify NXDOMAIN response
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	if mockWriter.msg.Rcode != dns.RcodeNameError {
		t.Errorf("Expected NXDOMAIN (rcode %d), got %d", dns.RcodeNameError, mockWriter.msg.Rcode)
	}

	if len(mockWriter.msg.Answer) != 0 {
		t.Errorf("Expected no answers for non-existent service, got %d", len(mockWriter.msg.Answer))
	}
}

func TestDNSServer_ExpiredService(t *testing.T) {
	// Use very short TTL for testing
	serviceStore := store.NewServiceStoreWithTTL(1 * time.Millisecond)
	log := logger.NewFromConfig("info", "text")

	config := Config{
		Host:   "localhost",
		Port:   0,
		Domain: "consul",
	}

	dnsServer := NewServer(config, serviceStore, log)

	// Register service
	service := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	serviceStore.Register(service)

	// Wait for service to expire
	time.Sleep(10 * time.Millisecond)

	// Query for expired service
	query := new(dns.Msg)
	query.SetQuestion("_web._tcp.service.consul.", dns.TypeSRV)

	mockWriter := &mockResponseWriter{}
	dnsServer.handleDNSRequest(mockWriter, query)

	// Verify NXDOMAIN response
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	if mockWriter.msg.Rcode != dns.RcodeNameError {
		t.Errorf("Expected NXDOMAIN for expired service, got rcode %d", mockWriter.msg.Rcode)
	}
}

func TestDNSServer_UnsupportedQueryType(t *testing.T) {
	dnsServer, serviceStore := setupTestServer()

	// Register test service
	service := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	serviceStore.Register(service)

	// Query for unsupported type (MX)
	query := new(dns.Msg)
	query.SetQuestion("web.service.consul.", dns.TypeMX)

	mockWriter := &mockResponseWriter{}
	dnsServer.handleDNSRequest(mockWriter, query)

	// Verify NXDOMAIN response for unsupported type
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	if mockWriter.msg.Rcode != dns.RcodeNameError {
		t.Errorf("Expected NXDOMAIN for unsupported query type, got rcode %d", mockWriter.msg.Rcode)
	}
}

func TestDNSServer_ANYQuery(t *testing.T) {
	dnsServer, serviceStore := setupTestServer()

	// Register test service
	service := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	serviceStore.Register(service)

	// Query for ANY type
	query := new(dns.Msg)
	query.SetQuestion("_web._tcp.service.consul.", dns.TypeANY)

	mockWriter := &mockResponseWriter{}
	dnsServer.handleDNSRequest(mockWriter, query)

	// Verify response contains both SRV and A records
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	// Should have at least SRV record(s)
	srvFound := false
	for _, answer := range mockWriter.msg.Answer {
		if _, ok := answer.(*dns.SRV); ok {
			srvFound = true
			break
		}
	}

	if !srvFound {
		t.Error("Expected SRV record in ANY query response")
	}
}

func TestDNSServer_MultipleServices(t *testing.T) {
	dnsServer, serviceStore := setupTestServer()

	// Register multiple services with same name
	service1 := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	service2 := store.Service{Name: "web", Address: "192.168.1.101", Port: 8080}
	service3 := store.Service{Name: "web", Address: "192.168.1.102", Port: 9090}

	serviceStore.Register(service1)
	serviceStore.Register(service2)
	serviceStore.Register(service3)

	// Query for SRV records
	query := new(dns.Msg)
	query.SetQuestion("_web._tcp.service.consul.", dns.TypeSRV)

	mockWriter := &mockResponseWriter{}
	dnsServer.handleDNSRequest(mockWriter, query)

	// Verify response
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	if len(mockWriter.msg.Answer) != 3 {
		t.Errorf("Expected 3 SRV records, got %d", len(mockWriter.msg.Answer))
	}

	// Verify different ports are returned
	ports := make(map[uint16]bool)
	for _, answer := range mockWriter.msg.Answer {
		if srv, ok := answer.(*dns.SRV); ok {
			ports[srv.Port] = true
		}
	}

	expectedPorts := []uint16{80, 8080, 9090}
	for _, port := range expectedPorts {
		if !ports[port] {
			t.Errorf("Expected port %d in SRV records", port)
		}
	}

	// Should have corresponding A records in additional section
	if len(mockWriter.msg.Extra) != 3 {
		t.Errorf("Expected 3 A records in additional section, got %d", len(mockWriter.msg.Extra))
	}
}

func TestDNSServer_InvalidDomainParsing(t *testing.T) {
	dnsServer, serviceStore := setupTestServer()

	// Register test service
	service := store.Service{Name: "web", Address: "192.168.1.100", Port: 80}
	serviceStore.Register(service)

	// Query with invalid format (too few parts)
	query := new(dns.Msg)
	query.SetQuestion("web.consul.", dns.TypeSRV)

	mockWriter := &mockResponseWriter{}
	dnsServer.handleDNSRequest(mockWriter, query)

	// Should return NXDOMAIN for invalid format
	if mockWriter.msg == nil {
		t.Fatal("Expected DNS response, got nil")
	}

	if mockWriter.msg.Rcode != dns.RcodeNameError {
		t.Errorf("Expected NXDOMAIN for invalid domain format, got rcode %d", mockWriter.msg.Rcode)
	}
}

// Mock response writer for testing
type mockResponseWriter struct {
	msg *dns.Msg
}

func (m *mockResponseWriter) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8600}
}

func (m *mockResponseWriter) RemoteAddr() net.Addr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (m *mockResponseWriter) WriteMsg(msg *dns.Msg) error {
	m.msg = msg
	return nil
}

func (m *mockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockResponseWriter) Close() error {
	return nil
}

func (m *mockResponseWriter) TsigStatus() error {
	return nil
}

func (m *mockResponseWriter) TsigTimersOnly(bool) {}

func (m *mockResponseWriter) Hijack() {}