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
		OptionUseTor(true),
	)

	for i := 0; i < 3; i++ {
		resp, err := h.Get("http://ipv4.icanhazip.com/")
		assert.Nil(t, err)
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			assert.Nil(t, err)
			resp.Body.Close()
			log.Debugf("resp: %s", bytes.TrimSpace(body))
		}
		resp, err = h.Get("https://httpstat.us/403")
		log.Debug(err)
		assert.Nil(t, err)
	}

	assert.Nil(t, h.Close())
}
