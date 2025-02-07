package resolvers

import "metadatatool/internal/graphql/generated"

// queryResolver implements the query resolver
type queryResolver struct {
	*generated.Resolver
}

// mutationResolver implements the mutation resolver
type mutationResolver struct {
	*generated.Resolver
}

// subscriptionResolver implements the subscription resolver
type subscriptionResolver struct {
	*generated.Resolver
}

// trackResolver implements the track resolver
type trackResolver struct {
	*generated.Resolver
}
