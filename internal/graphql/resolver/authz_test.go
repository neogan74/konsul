package resolver

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/auth"
	"github.com/neogan74/konsul/internal/logger"
)

func gqlContextWithAuthHeader(authHeader string) context.Context {
	opCtx := &graphql.OperationContext{
		Headers: make(http.Header),
	}
	if authHeader != "" {
		opCtx.Headers.Set("Authorization", authHeader)
	}
	return graphql.WithOperationContext(context.Background(), opCtx)
}

func TestClaimsFromGraphQLContext_NoJWTService(t *testing.T) {
	r := &Resolver{}

	claims, err := r.claimsFromGraphQLContext(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims != nil {
		t.Fatalf("expected nil claims, got %+v", claims)
	}
}

func TestClaimsFromGraphQLContext_ValidToken(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, time.Hour, "konsul-test")
	token, err := jwtService.GenerateToken("user1", "alice", []string{"user"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := &Resolver{jwtService: jwtService}
	ctx := gqlContextWithAuthHeader("Bearer " + token)

	claims, err := r.claimsFromGraphQLContext(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims == nil {
		t.Fatal("expected claims, got nil")
	}
	if claims.UserID != "user1" || claims.Username != "alice" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestClaimsFromGraphQLContext_Unauthorized(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, time.Hour, "konsul-test")
	r := &Resolver{jwtService: jwtService}

	_, err := r.claimsFromGraphQLContext(context.Background())
	if err == nil || err.Error() != "unauthorized" {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestAuthorizeMutation_ACL(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, time.Hour, "konsul-test")
	evaluator := acl.NewEvaluator(logger.GetDefault())

	policy := &acl.Policy{
		Name: "kv-write",
		KV: []acl.KVRule{
			{
				Path:         "config/*",
				Capabilities: []acl.Capability{acl.CapabilityWrite},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user1", "alice", []string{"user"}, []string{"kv-write"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := &Resolver{
		jwtService:   jwtService,
		aclEvaluator: evaluator,
	}

	allowCtx := gqlContextWithAuthHeader("Bearer " + token)
	if err := r.authorizeMutation(allowCtx, acl.NewKVResource("config/app"), acl.CapabilityWrite); err != nil {
		t.Fatalf("expected allow, got %v", err)
	}

	err = r.authorizeMutation(allowCtx, acl.NewKVResource("config/app"), acl.CapabilityDelete)
	if err == nil || err.Error() != "forbidden: insufficient permissions" {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestAuthorizeMutation_NoPolicies(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, time.Hour, "konsul-test")
	evaluator := acl.NewEvaluator(logger.GetDefault())

	token, err := jwtService.GenerateToken("user1", "alice", []string{"user"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := &Resolver{
		jwtService:   jwtService,
		aclEvaluator: evaluator,
	}
	ctx := gqlContextWithAuthHeader("Bearer " + token)

	err = r.authorizeMutation(ctx, acl.NewKVResource("config/app"), acl.CapabilityWrite)
	if err == nil || err.Error() != "forbidden: no policies attached to token" {
		t.Fatalf("expected missing policies error, got %v", err)
	}
}
