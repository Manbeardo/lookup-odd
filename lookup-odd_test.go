package lookupodd_test

import (
	"fmt"
	"math/rand"
	"testing"

	lookupodd "github.com/Manbeardo/lookup-odd"
	"github.com/stretchr/testify/assert"
)

func TestIsOdd(t *testing.T) {
	expectedValues := map[uint64]bool{
		0:                    false,
		1:                    true,
		2:                    false,
		3:                    true,
		11111111111111112:    false,
		18446744073709551615: true,
	}
	for num, expected := range expectedValues {
		t.Run(fmt.Sprintf("%d returns %v", num, expected), func(t *testing.T) {
			actual, err := lookupodd.IsOdd(num)
			assert.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}

func generateNumbers(count int) []uint64 {
	numbers := []uint64{}
	for i := 0; i < count; i++ {
		numbers = append(numbers, rand.Uint64())
	}
	return numbers
}

func BenchmarkIsOdd(b *testing.B) {
	b.StopTimer()
	numbers := generateNumbers(b.N)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lookupodd.IsOdd(numbers[i])
	}
}

func isOddFromModulus(num uint64) (bool, error) {
	return (num % 2) > 0, nil
}

func BenchmarkModulusOperator(b *testing.B) {
	b.StopTimer()
	numbers := generateNumbers(b.N)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = isOddFromModulus(numbers[i])
	}
}
