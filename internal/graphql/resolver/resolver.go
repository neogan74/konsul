package resolver

import (
	"time"

	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/auth"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
	"github.com/neogan74/konsul/internal/watch"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver is the root resolver
type Resolver struct {
	kvStore      *store.KVStore
	serviceStore *store.ServiceStore
	watchManager *watch.Manager
	aclEvaluator *acl.Evaluator
	jwtService   *auth.JWTService
	logger       logger.Logger
	version      string
	startTime    time.Time
}

// NewResolver creates a new resolver
func NewResolver(deps ResolverDependencies) *Resolver {
	return &Resolver{
		kvStore:      deps.KVStore,
		serviceStore: deps.ServiceStore,
		watchManager: deps.WatchManager,
		aclEvaluator: deps.ACLEvaluator,
		jwtService:   deps.JWTService,
		logger:       deps.Logger,
		version:      deps.Version,
		startTime:    time.Now(),
	}
}

// ResolverDependencies holds all dependencies for resolvers
type ResolverDependencies struct {
	KVStore      *store.KVStore
	ServiceStore *store.ServiceStore
	WatchManager *watch.Manager
	ACLEvaluator *acl.Evaluator
	JWTService   *auth.JWTService
	Logger       logger.Logger
	Version      string
}
