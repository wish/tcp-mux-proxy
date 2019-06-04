package loadbalancer

// LoadBalancer interface defines the functions
// the load balancer needs to implement
type LoadBalancer interface {
	// load balancer will need internal map(array) of ids to counts
	DecConn(id uint16) error
	IncConn(id uint16) error

	// proxy will call this to determine where to route request
	GetDownstream() uint16

	MarkHealthy(id uint16)
	MarkUnhealthy(id uint16)
}
