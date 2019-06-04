package loadbalancer

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestPowOfTwoLB(t *testing.T) {
	n := uint16(30)
	lb := NewPowerOfTwoLoadBalancer(n)
	record := make([]int, n)

	for i := 0; i < 3000000; i++ {
		id := lb.GetDownstream()
		record[id]++
		lb.IncConn(id)

		go func(id uint16, lb *PowerOfTwoLoadBalancer) {
			wait := time.Duration(int((1 + (rand.Float32()*2 - 1)) * 1000))
			time.Sleep(time.Millisecond * wait)
			lb.DecConn(id)
		}(id, lb)
	}

	sort.Ints(record)

	fmt.Println("Power of Two Load Balancer Test Results")
	fmt.Printf("Min: %v\n", record[0])
	fmt.Printf("25th: %v\n", int((record[6]+record[7])/2))
	fmt.Printf("50th: %v\n", record[14])
	fmt.Printf("75th: %v\n", int((record[21]+record[22])/2))
	fmt.Printf("Max: %v\n", record[29])
}

func TestLBAtomicIncDec(t *testing.T) {
	var wg sync.WaitGroup
	n := uint16(30)
	lb := NewPowerOfTwoLoadBalancer(n)

	wg.Add(100000)
	for i := 0; i < 100000; i++ {
		go func(lb *PowerOfTwoLoadBalancer) {
			defer wg.Done()
			id1 := uint16(rand.Intn(int(n)))
			id2 := uint16(rand.Intn(int(n)))

			lb.IncConn(id1)
			lb.IncConn(id2)

			wait := time.Duration(rand.Float32() * 2000)
			time.Sleep(time.Millisecond * wait)

			lb.DecConn(id1)
			lb.DecConn(id2)
		}(lb)
	}

	wg.Wait()

	for _, zero := range lb.getConnections() {
		if zero != 0 {
			t.Error("Some connection count was non-zero")
		}
	}
}

func TestBasicLBAtomicIncDec(t *testing.T) {
	n := uint16(30)
	lb := NewPowerOfTwoLoadBalancer(n)

	lb.IncConn(9)
	lb.IncConn(9)
	if lb.getConnections()[9] != 2 {
		t.Error("Connection count was not 2")
	}

	lb.DecConn(9)
	lb.DecConn(9)
	if lb.getConnections()[9] != 0 {
		t.Error("Connection count was not 0")
	}
}

//Inc and Dec both use atomic.AddUint32 so we just need to benchmark Inc
func BenchmarkBasicAtomicInc(b *testing.B) {
	n := uint16(30)
	lb := NewPowerOfTwoLoadBalancer(n)

	b.ResetTimer()

	for i := 0; i < 100000; i++ {
		id1 := uint16(rand.Intn(int(n)))
		lb.IncConn(id1)
	}
}

func BenchmarkContentionAtomicInc(b *testing.B) {
	n := uint16(30)
	lb := NewPowerOfTwoLoadBalancer(n)

	b.SetParallelism(10)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := uint16(rand.Intn(int(n)))
			lb.IncConn(id)
		}
	})
}
