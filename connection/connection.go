package connection

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/schollz/bine/tor"
	log "github.com/schollz/logger"
)

// Connection defines a connection
type Connection struct {
	// config paramters
	debug  bool
	name   string
	usetor bool

	connecting bool
	ready      bool
	readyLock  sync.Mutex
	ipAddress  string
	tor        *tor.Tor
	client     *http.Client
	timeout    time.Duration
	headers    map[string]string
}

var NotReadyError = errors.New("not ready")

// Option is the type all options need to adhere to
type Option func(c *Connection)

// OptionDebug turns on debugging
func OptionDebug(debug bool) Option {
	return func(c *Connection) {
		c.debug = debug
	}
}

// OptionUseTor turns on debugging
func OptionUseTor(usetor bool) Option {
	return func(c *Connection) {
		c.usetor = usetor
	}
}

// OptionTimeout sets timeout
func OptionTimeout(timeout time.Duration) Option {
	return func(c *Connection) {
		c.timeout = timeout
	}
}

// OptionDebug turns on debugging
func OptionHeaders(headers map[string]string) Option {
	return func(c *Connection) {
		c.headers = make(map[string]string)
		for h := range headers {
			c.headers[h] = headers[h]
		}
	}
}

// OptionName turns on debugging
func OptionName(name string) Option {
	return func(c *Connection) {
		c.name = name
	}
}

// New constructs a new instance of HTTPPool
func New(options ...Option) *Connection {
	c := Connection{
		debug:   false,
		usetor:  true,
		timeout: 180 * time.Second,
		name:    "", // TOOD: use UUID?
		headers: make(map[string]string),
	}
	for _, o := range options {
		o(&c)
	}

	return &c
}

// Close will close connections
func (c *Connection) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Trace("Recovered in Close", r)
		}
	}()

	c.ready = false
	if c.client != nil {
		c.client.CloseIdleConnections()
	}
	if c.tor != nil {
		err = c.tor.Close()
	}
	return
}

// Connect will connect
func (c *Connection) Connect() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Trace("Recovered in Connect", r)
		}
	}()

	if c.connecting {
		return
	}
	c.connecting = true
	c.ready = false

	// close any current connections
	c.Close()

	log.Tracef("%s setting up client", c.name)
	c.client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 30,
		},
		Timeout: c.timeout,
	}

	// keep trying until it gets on
	for {
		if !c.usetor {
			break
		}
		log.Debug("connecting to tor...")
		c.tor, err = tor.Start(nil, nil)
		if err != nil {
			log.Error(err)
			continue
		}

		// Wait at most a minute to start network and get
		dialCtx, dialCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer dialCancel()
		// Make connection
		dialer, err := c.tor.Dialer(dialCtx, nil)
		if err != nil {
			log.Debug(err)
			continue
		}
		c.client.Transport = &http.Transport{
			DialContext:         dialer.DialContext,
			MaxIdleConnsPerHost: 20,
		}
		err = c.getPublicIP()
		if err != nil {
			continue
		}
		log.Tracef("new IP address: %s", c.ipAddress)
		break
	}

	c.ready = true
	c.connecting = false
	return
}

// PublicIP returns the current public IP address, getting it if hasn't been set
func (c *Connection) PublicIP() string {
	if c.ipAddress == "" {
		c.getPublicIP()
	}
	return c.ipAddress
}

func (c *Connection) getPublicIP() (err error) {
	resp, err := c.client.Get("http://icanhazip.com/")
	if err != nil {
		log.Debug(err)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Debug(err)
		return
	}
	c.ipAddress = string(bytes.TrimSpace(body))
	return
}

// Get will get a URL
func (c *Connection) Get(urlToGet string) (resp *http.Response, err error) {
	if !c.ready {
		err = NotReadyError
		return
	}

	log.Debugf("[%s] getting %s", c.name, urlToGet)
	req, err := http.NewRequest("GET", urlToGet, nil)
	if err != nil {
		return
	}
	for h := range c.headers {
		req.Header.Set(h, c.headers[h])
	}
	resp, err = c.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if err != nil {
			log.Tracef("[%s] got error: %s", c.name, err.Error())
		} else {
			log.Tracef("[%s] got status code: %d", c.name, resp.StatusCode)
			// bad code received, reload
			go c.Connect()
		}
	}

	return
}
