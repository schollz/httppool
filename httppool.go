package httppool

import (
	"fmt"
	"sync"

	"github.com/schollz/httppool/connection"
	log "github.com/schollz/logger"
)

// HTTPPool defines the pool of HTTP requests
type HTTPPool struct {
	debug      bool
	numClients int

	conn []*connection.Connection
}

// Option is the type all options need to adhere to
type Option func(h *HTTPPool)

// OptionDebug turns on debugging
func OptionDebug(debug bool) Option {
	return func(h *HTTPPool) {
		h.debug = debug
	}
}

// OptionNumClients sets the number of clients
func OptionNumClients(num int) Option {
	return func(h *HTTPPool) {
		h.numClients = num
	}
}

// New constructs a new instance of HTTPPool
func New(options ...Option) *HTTPPool {
	h := HTTPPool{
		debug:      false,
		numClients: 2,
	}
	for _, o := range options {
		o(&h)
	}

	if h.debug {
		log.SetLevel("debug")
		log.Debug("debug mode on")
	} else {
		log.SetLevel("info")
	}

	h.conn = make([]*connection.Connection, h.numClients)
	var wg sync.WaitGroup
	wg.Add(h.numClients)
	for i := 0; i < h.numClients; i++ {
		go func(i int) {
			defer wg.Done()
			h.conn[i] = connection.New(
				connection.OptionDebug(h.debug),
				connection.OptionName(fmt.Sprintf("%d", i)),
			)
			h.conn[i].Connect()
		}(i)
	}
	wg.Wait()
	return &h
}

// Close shuts down any nodes
func (h *HTTPPool) Close() (err error) {
	for i := 0; i < h.numClients; i++ {
		err = h.conn[i].Close()
		if err != nil {
			log.Warn(err)
		}
	}
	return
}
