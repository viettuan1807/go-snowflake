package snowflake_test

import (
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