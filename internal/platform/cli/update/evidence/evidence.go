package evidence

// Kind identifies an observable update lifecycle or release update event.
type Kind string

// Item is one operator-visible piece of update evidence.
type Item struct {
	Kind   Kind
	Detail string
}
