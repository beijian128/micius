package appframeslb

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestQuickSelect(t *testing.T) {
	arr := []uint32{11, 12, 13, 14, 15, 16}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(arr), func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})
	k1 := QuickSelect(arr, 1)
	t.Log(arr, k1)
	assert.Equal(t, uint32(11), QuickSelect(arr, 0))
	assert.Equal(t, uint32(12), QuickSelect(arr, 1))
	assert.Equal(t, uint32(13), QuickSelect(arr, 2))
	assert.Equal(t, uint32(14), QuickSelect(arr, 3))
	assert.Equal(t, uint32(15), QuickSelect(arr, 4))
	assert.Equal(t, uint32(16), QuickSelect(arr, 5))

}
