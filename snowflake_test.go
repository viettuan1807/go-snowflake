package snowflake_test

import (
	"sync"
	"testing"

	"github.com/viettuan1807/go-snowflake"
)

func BenchmarkSnowflake(b *testing.B) {
	sf := snowflake.NewSnowflake(12)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.NextID()
	}
}

func BenchmarkSnowflakeParallel(b *testing.B) {
	sf := snowflake.NewSnowflake(12)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = sf.NextID()
		}
	})
}

func BenchmarkGetTimestamp(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	id := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.GetTimestamp(id)
	}
}

func BenchmarkGetWorkerID(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	id := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.GetWorkerID(id)
	}
}

func BenchmarkGetDatacenterID(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	id := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.GetDatacenterID(id)
	}
}

func BenchmarkGetSequence(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	id := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.GetSequence(id)
	}
}

func BenchmarkConcurrentGeneration(b *testing.B) {
	sf := snowflake.NewSnowflake(12)
	numWorkers := 10

	b.ResetTimer()
	var wg sync.WaitGroup
	idsPerWorker := b.N / numWorkers

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerWorker; j++ {
				_ = sf.NextID()
			}
		}()
	}
	wg.Wait()
}
