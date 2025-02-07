package resolvers

import (
	"metadatatool/internal/graphql/generated"
)

// Resolver is the base resolver for all GraphQL operations
type Resolver struct {
	*generated.Resolver
}

// NewResolver creates a new resolver instance
func NewResolver(r *generated.Resolver) *Resolver {
	return &Resolver{
		Resolver: r,
	}
}

// Query returns the query resolver
func (r *Resolver) Query() generated.QueryResolver {
	return &queryResolver{r.Resolver}
}

// Mutation returns the mutation resolver
func (r *Resolver) Mutation() generated.MutationResolver {
	return &mutationResolver{r.Resolver}
}

// Subscription returns the subscription resolver
func (r *Resolver) Subscription() generated.SubscriptionResolver {
	return &subscriptionResolver{r.Resolver}
}

// Track returns the track resolver
func (r *Resolver) Track() generated.TrackResolver {
	return &trackResolver{r.Resolver}
}
