package httppool

import (
	"bytes"
	"io/ioutil"
	"testing"

	log "github.com/schollz/logger"
	"github.com/stretchr/testify/assert"
)

func TestHTTPPool(t *testing.T) {
	h := New(
		OptionDebug(true),
		OptionNumClients(2),
	)

	for i := 0; i < 3; i++ {
		resp, err := h.Get("https://httpstat.us/403")
		log.Debug(err)
		assert.Nil(t, err)
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			assert.Nil(t, err)
			resp.Body.Close()
			log.Debugf("IP: %s", bytes.TrimSpace(body))
		}

	}

	assert.Nil(t, h.Close())
}
