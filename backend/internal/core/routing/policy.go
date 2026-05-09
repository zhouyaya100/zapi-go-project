package routing

type RoutingStrategy int

const (
	PriorityWeighted RoutingStrategy = iota
	RoundRobin
	LeastLatency
)

type RoutingPolicy struct{ rrCounter map[string]int }

var Policy = &RoutingPolicy{rrCounter: make(map[string]int)}
