package model

// Entity represents an object defined by its identity rather than attributes.
type Entity interface {
	EntityID() string
}

// AggregateRoot represents the entrypoint and boundary of a domain aggregate.
type AggregateRoot interface {
	Entity
	AggregateType() string
}
