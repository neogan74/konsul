package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/neogan74/konsul/internal/graphql/generated"
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

	// Create GraphQL handler
	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{
				Resolvers: r,
			},
		),
	)

	// Add middleware (will expand in Phase 3)
	// srv.Use(extension.FixedComplexityLimit(1000))

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
