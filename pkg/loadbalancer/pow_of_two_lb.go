package loadbalancer

import (
	"errors"
	"math/rand"
	"sync/atomic"
)

// PowerOfTwoLoadBalancer implementation
type PowerOfTwoLoadBalancer struct {
	// number of ports
	n           uint16
	connections []uint32
	idUnhealthy []uint32
}

// NewPowerOfTwoLoadBalancer makes a PowerOfTwoLoadBalancer and returns it
func NewPowerOfTwoLoadBalancer(n uint16) *PowerOfTwoLoadBalancer {
	return &PowerOfTwoLoadBalancer{
		n:           n,
		connections: make([]uint32, n),
		idUnhealthy: make([]uint32, n),
	}
}

// MarkHealthy allows health monitor to tell LB when a downstream id becomes healthy
func (lb *PowerOfTwoLoadBalancer) MarkHealthy(id uint16) {
	atomic.StoreUint32(&lb.idUnhealthy[id], 0)
}

// MarkUnhealthy allows health monitor to tell LB when a downstream id becomes unhealthy
func (lb *PowerOfTwoLoadBalancer) MarkUnhealthy(id uint16) {
	atomic.StoreUint32(&lb.idUnhealthy[id], 1)
}

// DecConn decrements the number of connections of a particular downstream
func (lb *PowerOfTwoLoadBalancer) DecConn(id uint16) error {
	if id < 0 || id >= lb.n {
		return errors.New("Invalid id")
	}
	if atomic.LoadUint32(&lb.connections[id]) <= 0 {
		return errors.New("Count cannot be less than zero")
	}
	// this will decrement the value: https:// golang.org/pkg/sync/atomic/#AddUint32
	atomic.AddUint32(&lb.connections[id], ^uint32(0))
	return nil
}

// IncConn increments the number of connections of a particular downstream
func (lb *PowerOfTwoLoadBalancer) IncConn(id uint16) error {
	if id < 0 || id >= lb.n {
		return errors.New("Invalid id")
	}

	atomic.AddUint32(&lb.connections[id], uint32(1))
	return nil
}

// GetDownstream uses the power of two algorithm to determine which
// connection to forward request to, returns the id
func (lb *PowerOfTwoLoadBalancer) GetDownstream() uint16 {
	var id uint16
	// limit the number of failed attempts, if we fail numerous times
	// server likely in shutdown anyways
	// wanted to have some stopping condition here - not sure if this is best one
	for i := 0; i < 5; i++ {
		id1 := uint16(rand.Intn(int(lb.n)))
		id2 := uint16(rand.Intn(int(lb.n)))

		if id1 == id2 {
			id2 = (id2 + lb.n/2) % lb.n
		}

		if atomic.LoadUint32(&lb.connections[id1]) > atomic.LoadUint32(&lb.connections[id2]) {
			id = id2
		} else {
			id = id1
		}
		if atomic.LoadUint32(&lb.idUnhealthy[id]) == 0 {
			break
		}
	}
	return id
}

// this should only be used for testing
func (lb *PowerOfTwoLoadBalancer) getConnections() []uint32 {
	connectionsCopy := make([]uint32, len(lb.connections))
	copy(connectionsCopy, lb.connections)
	return connectionsCopy
}
