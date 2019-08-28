package httppool

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/schollz/httppool/connection"
	log "github.com/schollz/logger"
)

// HTTPPool defines the pool of HTTP requests
type HTTPPool struct {
	debug      bool
	numClients int
	usetor     bool
	headers    map[string]string

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

// OptionuseTor turns on debugging
func OptionUseTor(usetor bool) Option {
	return func(h *HTTPPool) {
		h.usetor = usetor
	}
}

// OptionNumClients sets the number of clients
func OptionNumClients(num int) Option {
	return func(h *HTTPPool) {
		h.numClients = num
	}
}

// OptionHeaders turns on debugging
func OptionHeaders(headers map[string]string) Option {
	return func(h *HTTPPool) {
		h.headers = make(map[string]string)
		for header := range headers {
			h.headers[header] = headers[header]
		}
	}
}

// New constructs a new instance of HTTPPool
func New(options ...Option) *HTTPPool {
	h := HTTPPool{
		debug:      false,
		numClients: 2,
		headers:    make(map[string]string),
	}
	for _, o := range options {
		o(&h)
	}

	if h.debug {
		log.SetLevel("trace")
		log.Trace("debug mode on")
	} else {
		log.SetLevel("info")
	}

	h.conn = make([]*connection.Connection, h.numClients)
	for i := 0; i < h.numClients; i++ {
		h.conn[i] = connection.New(
			connection.OptionDebug(h.debug),
			connection.OptionName(fmt.Sprintf("%d", i)),
			connection.OptionHeaders(h.headers),
			connection.OptionUseTor(h.usetor),
		)
		log.Tracef("starting connection for %d", i)
		go h.conn[i].Connect()
	}

	return &h
}

// Close shuts down any nodes
func (h *HTTPPool) Close() (err error) {
	for i := 0; i < h.numClients; i++ {
		err = h.conn[i].Close()
		if err != nil {
			log.Trace("close error", err)
		}
	}
	return
}

// Get will randomly select a client in the pool
func (h *HTTPPool) Get(urlToGet string) (resp *http.Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()

	ar := make([]int, h.numClients)
	for i := 0; i < len(ar); i++ {
		ar[i] = i
	}

	// try one until we get one that is ready
tryagain:
	shuffle(ar)
	for _, i := range ar {
		resp, err = h.conn[i].Get(urlToGet)
		if err != nil {
			switch err {
			case connection.NotReadyError:
				continue
			default:
				break
			}
		}
		break
	}
	if err != nil {
		switch err {
		case connection.NotReadyError:
			time.Sleep(1 * time.Second)
			goto tryagain
		default:
			log.Trace("Unknown error occurred")
		}
	}
	return
}

func shuffle(slice []int) {
	for len(slice) > 0 {
		n := len(slice)
		randIndex := rand.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}
