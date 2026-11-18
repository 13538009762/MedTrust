package service

import (
	"fmt"
	"sync"
	"testing"
)

func TestConsistency_KeyLocksReleasedAfterUnlock(t *testing.T) {
	svc := &ConsistencyService{}
	const goroutines = 32
	const keysPerG = 8

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for k := 0; k < keysPerG; k++ {
				// Mix shared keys (contention) and unique keys (simulating empty idempotencyKey)
				key := fmt.Sprintf("shared-%d", k%3)
				if k%2 == 1 {
					key = fmt.Sprintf("anon:%d:%d", gid, k)
				}
				unlock := svc.lockKey(key)
				unlock()
			}
		}(g)
	}
	wg.Wait()

	if count := svc.KeyLockCount(); count != 0 {
		t.Fatalf("expected all keyLocks entries to be released after unlock, got %d remaining", count)
	}
}
