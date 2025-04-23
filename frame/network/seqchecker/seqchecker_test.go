package seqchecker

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_absDeltaTime(t *testing.T) {
	t.Log(absDeltaTime(1692009325, 1692009326))
	assert.Equal(t, absDeltaTime(1692009325, 1692009326), time.Second)
	assert.Equal(t, absDeltaTime(1692009326, 1692009325), time.Second)
}
