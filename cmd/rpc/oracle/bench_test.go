//go:build exclude

package oracle

import (
	"os"
	"testing"

	// "github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
)

func BenchmarkOracleDiskStorage(b *testing.B) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "oracle_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create storage instance
	storage, err := NewOracleDiskStorage(tempDir, lib.NewDefaultLogger())
	if err != nil {
		b.Fatal(err)
	}

	// Create test data
	orderId := make([]byte, 20) // 20 bytes as per orderIdLenBytes
	for i := range orderId {
		orderId[i] = byte(i)
	}

	// orderType := types.LockOrderType

	b.Run("WriteReadOrder", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create unique order ID for each iteration
			testOrderId := make([]byte, 20)
			copy(testOrderId, orderId)
			testOrderId[19] = byte(i % 256)
			testOrderId[18] = byte(i >> 8)

			// Create test witnessed order
			// witnessedOrder := &types.WitnessedOrder{
			// 	OrderId:          testOrderId,
			// 	WitnessedHeight:  uint64(i),
			// 	LastSubmitHeight: uint64(i),
			// 	LockOrder:        &lib.LockOrder{},
			// 	CloseOrder:       &lib.CloseOrder{},
			// }

			// err := storage.WriteOrder(witnessedOrder, orderType)
			// if err != nil {
			// 	b.Fatal(err)
			// }

			// _, err = storage.ReadOrder(testOrderId, orderType)
			// if err != nil {
			// 	b.Fatal(err)
			// }
			// err = storage.RemoveOrder(testOrderId, orderType)
			// if err != nil {
			// 	b.Fatal(err)
			// }
		}
	})
}
