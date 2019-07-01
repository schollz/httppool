package httppool

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPPool(t *testing.T) {
	h := New(OptionDebug(true))
	fmt.Println(h)
	assert.Nil(t, nil)

	assert.Nil(t, h.Close())
}
