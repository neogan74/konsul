package resolver

import (
	"context"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"

	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/auth"
)

// claimsFromGraphQLContext validates JWT from GraphQL operation headers.
func (r *Resolver) claimsFromGraphQLContext(ctx context.Context) (*auth.Claims, error) {
	if r.jwtService == nil {
		return nil, nil
	}

	if !graphql.HasOperationContext(ctx) {
		return nil, fmt.Errorf("unauthorized")
	}

	opCtx := graphql.GetOperationContext(ctx)
	authHeader := opCtx.Headers.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("unauthorized")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, fmt.Errorf("unauthorized")
	}

	claims, err := r.jwtService.ValidateToken(parts[1])
	if err != nil {
		return nil, fmt.Errorf("unauthorized")
	}

	return claims, nil
}

// authorizeMutation enforces auth and ACL checks for GraphQL mutation operations.
func (r *Resolver) authorizeMutation(ctx context.Context, resource acl.Resource, capability acl.Capability) error {
	claims, err := r.claimsFromGraphQLContext(ctx)
	if err != nil {
		return err
	}

	// ACL checks are only enforced when ACL evaluator is configured.
	if r.aclEvaluator == nil {
		return nil
	}

	if claims == nil {
		return fmt.Errorf("unauthorized")
	}

	if len(claims.Policies) == 0 {
		return fmt.Errorf("forbidden: no policies attached to token")
	}

	if !r.aclEvaluator.Evaluate(claims.Policies, resource, capability) {
		return fmt.Errorf("forbidden: insufficient permissions")
	}

	return nil
}
