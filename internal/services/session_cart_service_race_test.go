package services

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// SYS-01 regression: concurrent access to the in-memory sessionCarts map must not
// trigger "fatal error: concurrent map read and map write". Run with -race to also
// catch data races on the map. Before the fix (unguarded map) this panics the
// process; with the mutex-guarded helpers it is safe.
func TestSessionCartService_ConcurrentMapAccess(t *testing.T) {
	s := &SessionCartService{
		sessionCarts: make(map[string]*models.Cart),
	}

	// Enough concurrent goroutines with overlapping keys to reliably interleave
	// reads, writes and deletes on the shared map (which panics the runtime if the
	// map is unguarded). Kept small: with zero work between lock ops this is pure
	// mutex contention, and on platforms with coarse semaphore wakeups a large op
	// count degrades into a slow lock convoy without adding coverage.
	const goroutines = 4
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			sid := fmt.Sprintf("sess-%d", g%8) // deliberate key overlap across goroutines
			for i := 0; i < iterations; i++ {
				switch i % 3 {
				case 0:
					s.storeCart(sid, &models.Cart{})
				case 1:
					_, _ = s.loadCart(sid)
				case 2:
					s.deleteCart(sid)
				}
			}
		}(g)
	}
	wg.Wait()
}
