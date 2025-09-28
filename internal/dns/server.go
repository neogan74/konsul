package dns

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

type Server struct {
	udpServer *dns.Server
	tcpServer *dns.Server
	domain    string
	store     *store.ServiceStore
	log       logger.Logger
}

type Config struct {
	Host   string
	Port   int
	Domain string
}

func NewServer(cfg Config, serviceStore *store.ServiceStore, log logger.Logger) *Server {
	s := &Server{
		domain: cfg.Domain,
		store:  serviceStore,
		log:    log,
	}

	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handleDNSRequest)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	s.udpServer = &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: mux,
	}

	s.tcpServer = &dns.Server{
		Addr:    addr,
		Net:     "tcp",
		Handler: mux,
	}

	return s
}

func (s *Server) Start() error {
	s.log.Info("Starting DNS server",
		logger.String("domain", s.domain),
		logger.String("udp_addr", s.udpServer.Addr),
		logger.String("tcp_addr", s.tcpServer.Addr))

	// Start UDP server
	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			s.log.Error("DNS UDP server failed", logger.Error(err))
		}
	}()

	// Start TCP server
	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			s.log.Error("DNS TCP server failed", logger.Error(err))
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	var udpErr, tcpErr error

	if s.udpServer != nil {
		udpErr = s.udpServer.Shutdown()
	}

	if s.tcpServer != nil {
		tcpErr = s.tcpServer.Shutdown()
	}

	if udpErr != nil {
		return udpErr
	}
	return tcpErr
}

func (s *Server) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	for _, question := range r.Question {
		s.log.Debug("DNS query received",
			logger.String("name", question.Name),
			logger.String("type", dns.TypeToString[question.Qtype]))

		switch question.Qtype {
		case dns.TypeSRV:
			s.handleSRVQuery(msg, question)
		case dns.TypeA:
			s.handleAQuery(msg, question)
		case dns.TypeANY:
			s.handleSRVQuery(msg, question)
			s.handleAQuery(msg, question)
		default:
			s.log.Debug("Unsupported DNS query type",
				logger.String("type", dns.TypeToString[question.Qtype]))
		}
	}

	if len(msg.Answer) == 0 {
		msg.Rcode = dns.RcodeNameError
	}

	w.WriteMsg(msg)
}

func (s *Server) handleSRVQuery(msg *dns.Msg, question dns.Question) {
	name := strings.TrimSuffix(question.Name, ".")

	// Parse SRV query: _service._protocol.service.consul
	parts := strings.Split(name, ".")
	if len(parts) < 4 {
		return
	}

	serviceName := strings.TrimPrefix(parts[0], "_")
	protocol := strings.TrimPrefix(parts[1], "_")

	// For now, we'll ignore protocol and just match service name
	_ = protocol

	// Get all healthy services matching the name
	services := s.store.List()
	var matchingServices []store.Service

	for _, service := range services {
		if service.Name == serviceName {
			matchingServices = append(matchingServices, service)
		}
	}

	// Create SRV records
	for i, service := range matchingServices {
		target := fmt.Sprintf("%s.node.%s.", service.Name, s.domain)

		srv := &dns.SRV{
			Hdr: dns.RR_Header{
				Name:   question.Name,
				Rrtype: dns.TypeSRV,
				Class:  dns.ClassINET,
				Ttl:    30,
			},
			Priority: 1,
			Weight:   uint16(100 / (i + 1)), // Simple weight distribution
			Port:     uint16(service.Port),
			Target:   target,
		}
		msg.Answer = append(msg.Answer, srv)

		// Add corresponding A record in Additional section
		a := &dns.A{
			Hdr: dns.RR_Header{
				Name:   target,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    30,
			},
			A: net.ParseIP(service.Address),
		}
		msg.Extra = append(msg.Extra, a)
	}

	s.log.Debug("SRV query processed",
		logger.String("service", serviceName),
		logger.Int("matches", len(matchingServices)))
}

func (s *Server) handleAQuery(msg *dns.Msg, question dns.Question) {
	name := strings.TrimSuffix(question.Name, ".")

	// Parse A query: service.node.consul or service-name.service.consul
	parts := strings.Split(name, ".")
	if len(parts) < 3 {
		return
	}

	var serviceName string

	// Check if it's a node query (service.node.consul)
	if len(parts) >= 3 && parts[1] == "node" {
		serviceName = parts[0]
	} else if len(parts) >= 3 && parts[1] == "service" {
		// service-name.service.consul format
		serviceName = parts[0]
	} else {
		return
	}

	// Get all healthy services matching the name
	services := s.store.List()

	for _, service := range services {
		if service.Name == serviceName {
			a := &dns.A{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    30,
				},
				A: net.ParseIP(service.Address),
			}
			msg.Answer = append(msg.Answer, a)
		}
	}

	s.log.Debug("A query processed",
		logger.String("service", serviceName),
		logger.Int("records", len(msg.Answer)))
}