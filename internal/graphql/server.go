package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/neogan74/konsul/internal/graphql/generated"
	"github.com/neogan74/konsul/internal/graphql/middleware"
	"github.com/neogan74/konsul/internal/graphql/resolver"
)

// Server wraps the GraphQL handler
type Server struct {
	handler    http.Handler
	playground http.Handler
}

// NewServer creates a new GraphQL server
func NewServer(deps resolver.ResolverDependencies) *Server {
	// Create resolver
	r := resolver.NewResolver(deps)

	// Create GraphQL schema
	schema := generated.NewExecutableSchema(
		generated.Config{
			Resolvers: r,
		},
	)

	// Create GraphQL handler with WebSocket support
	srv := handler.New(schema)

	// Enable WebSocket transport for subscriptions
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10, // Send ping every 10 seconds
	})

	// Enable HTTP POST transport for queries/mutations
	srv.AddTransport(transport.POST{})

	// Enable HTTP GET transport for queries (optional)
	srv.AddTransport(transport.GET{})

	// Phase 3: Add query complexity limits
	// Prevent expensive queries that could DoS the server
	srv.Use(extension.FixedComplexityLimit(1000))

	// Phase 3: Add query depth limiting
	// Prevent deeply nested queries (max 10 levels)
	srv.AroundOperations(middleware.DepthLimit(10))

	// Phase 3: Add introspection (enabled by default, can be disabled in production)
	srv.Use(extension.Introspection{})

	return &Server{
		handler:    srv,
		playground: playground.Handler("GraphQL Playground", "/graphql"),
	}
}

// Handler returns the GraphQL HTTP handler
func (s *Server) Handler() http.Handler {
	return s.handler
}

// PlaygroundHandler returns the GraphiQL playground handler
func (s *Server) PlaygroundHandler() http.Handler {
	return s.playground
}
