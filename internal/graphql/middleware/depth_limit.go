package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// DepthLimit creates a middleware that limits query depth
// This prevents deeply nested queries that could cause performance issues
func DepthLimit(maxDepth int) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)

		// Calculate query depth
		depth := calculateDepth(oc.Operation.SelectionSet, 0)

		if depth > maxDepth {
			return func(ctx context.Context) *graphql.Response {
				return graphql.ErrorResponse(ctx, "query depth %d exceeds maximum allowed depth of %d", depth, maxDepth)
			}
		}

		return next(ctx)
	}
}

// calculateDepth recursively calculates the maximum depth of a selection set
func calculateDepth(selectionSet ast.SelectionSet, currentDepth int) int {
	if len(selectionSet) == 0 {
		return currentDepth
	}

	maxDepth := currentDepth
	for _, selection := range selectionSet {
		switch sel := selection.(type) {
		case *ast.Field:
			// Skip __typename and other introspection fields
			if sel.Name == "__typename" || sel.Name == "__type" || sel.Name == "__schema" {
				continue
			}

			// Calculate depth for this field's selection set
			fieldDepth := calculateDepth(sel.SelectionSet, currentDepth+1)
			if fieldDepth > maxDepth {
				maxDepth = fieldDepth
			}

		case *ast.FragmentSpread:
			// Handle fragment spreads
			if sel.Definition != nil {
				fragmentDepth := calculateDepth(sel.Definition.SelectionSet, currentDepth)
				if fragmentDepth > maxDepth {
					maxDepth = fragmentDepth
				}
			}

		case *ast.InlineFragment:
			// Handle inline fragments
			fragmentDepth := calculateDepth(sel.SelectionSet, currentDepth)
			if fragmentDepth > maxDepth {
				maxDepth = fragmentDepth
			}
		}
	}

	return maxDepth
}
