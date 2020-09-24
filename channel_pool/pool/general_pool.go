package pool

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jasonkayzk/pool/channel_pool/errs"
	log "github.com/sirupsen/logrus"
)

// Configs for pool
type Options struct {
	// The number of the connections when initiate the pool
	// Also, the least connection number of the pool
	InitialCap int

	// Max connection number in the pool
	MaxCap int

	// Max idle number in the pool
	MaxIdle int

	// The method the build the connection
	Factory func() (interface{}, error)

	// The method to close the connection
	Close func(interface{}) error

	// Check connection health
	Ping func(interface{}) error

	// Max life time for idle connection
	IdleTimeout time.Duration

	// Max time to get a connection from pool
	// Else this will return a errs.MaxActiveConnectionErr
	WaitTimeout time.Duration
}

// the pool
type channelPool struct {
	mu           sync.RWMutex
	conns        chan *generalConn
	factory      func() (interface{}, error)
	close        func(interface{}) error
	ping         func(interface{}) error
	idleTimeout  time.Duration
	waitTimeOut  time.Duration
	maxActive    int
	openingConns int
	connReqs     []chan *generalConn
}

// Build pool
func NewChannelPool(options *Options) (Pool, error) {
	if !(options.InitialCap <= options.MaxIdle && options.MaxCap >= options.MaxIdle && options.InitialCap >= 0) {
		return nil, errors.New("invalid capacity settings")
	}
	if options.Factory == nil {
		return nil, errors.New("invalid factory func settings")
	}
	if options.Close == nil {
		return nil, errors.New("invalid close func settings")
	}
	if options.WaitTimeout <= 0 {
		options.WaitTimeout = time.Second * 3
	}

	cp := &channelPool{
		conns:        make(chan *generalConn, options.MaxIdle),
		factory:      options.Factory,
		close:        options.Close,
		idleTimeout:  options.IdleTimeout,
		maxActive:    options.MaxCap,
		openingConns: options.InitialCap,

	}

	if options.Ping != nil {
		cp.ping = options.Ping
	}

	for i := 0; i < options.InitialCap; i++ {
		conn, err := cp.factory()
		if err != nil {
			if err := cp.ShutDown(); err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("fill pool err: %s", err)
		}
		cp.conns <- &generalConn{conn: conn, t: time.Now()}
	}

	return cp, nil
}

func (c *channelPool) Get() (interface{}, error) {
	conns := c.getConns()
	if conns == nil {
		return nil, errs.NewDefaultClosedErr()
	}

	for {
		select {
		case wrapConn := <-conns:
			{
				if wrapConn == nil {
					return nil, errs.NewDefaultClosedErr()
				}

				// check timeout, if timeout remove
				if timeout := c.idleTimeout; timeout > 0 {
					if wrapConn.t.Add(timeout).Before(time.Now()) {
						//丢弃并关闭该连接
						c.CloseConn(wrapConn.conn)
						continue
					}
				}
				// check health, if not health remove
				// if no ping method, pass
				if c.ping != nil {
					if err := c.Ping(wrapConn.conn); err != nil {
						c.CloseConn(wrapConn.conn)
						continue
					}
				}
				return wrapConn.conn, nil
			}
		default:
			{
				c.mu.Lock()
				log.Debugf("openConn %v %v", c.openingConns, c.maxActive)
				if c.openingConns >= c.maxActive {
					req := make(chan *generalConn, 1)
					c.connReqs = append(c.connReqs, req)
					c.mu.Unlock()

					select {
					case ret, ok := <-req:
						{
							if !ok {
								return nil, errs.NewMaxActiveConnectionErr("max active connection limit")
							}
							if timeout := c.idleTimeout; timeout > 0 {
								if ret.t.Add(timeout).Before(time.Now()) {
									// check timeout, if timeout remove
									c.CloseConn(ret.conn)
									continue
								}
							}
							return ret.conn, nil
						}
					case <-time.After(c.waitTimeOut):
						return nil, errs.NewWaitConnectionTimeoutErr("get active connection timeout")
					}
				}
				if c.factory == nil {
					c.mu.Unlock()
					return nil, errs.NewDefaultClosedErr()
				}

				conn, err := c.factory()
				if err != nil {
					c.mu.Unlock()
					return nil, err
				}
				c.openingConns++
				c.mu.Unlock()
				return conn, nil
			}
		}
	}
}

func (c *channelPool) Put(conn interface{}) error {
	if conn == nil {
		return errors.New("nil connection err")
	}

	c.mu.Lock()

	// single coroutine pool
	if c.conns == nil {
		c.mu.Unlock()
		return c.CloseConn(conn)
	}

	//
	if l := len(c.connReqs); l > 0 {
		req := c.connReqs[0]
		copy(c.connReqs, c.connReqs[1:])
		c.connReqs = c.connReqs[:l-1]
		req <- &generalConn{
			conn: conn,
			t:    time.Now(),
		}
		c.mu.Unlock()
		return nil
	} else {
		select {
		case c.conns <- &generalConn{
			conn: conn, t:
			time.Now(),
		}:
			c.mu.Unlock()
			return nil
		default:
			c.mu.Unlock()
			// pool is full, close pool directly
			return c.CloseConn(conn)
		}
	}
}

func (c *channelPool) CloseConn(conn interface{}) error {
	if conn == nil {
		return errors.New("nil connection err")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.close == nil {
		return nil
	}

	c.openingConns--
	return c.close(conn)
}

// Ping check connection health
func (c *channelPool) Ping(conn interface{}) error {
	if conn == nil {
		return errors.New("nil connection err")
	}
	return c.ping(conn)
}

func (c *channelPool) ShutDown() error {
	c.mu.Lock()
	conns := c.conns
	c.conns = nil
	c.factory = nil
	c.ping = nil
	closeFun := c.close
	c.close = nil
	c.mu.Unlock()

	if conns == nil {
		return nil
	}

	close(conns)
	for wrapConn := range conns {
		if err := closeFun(wrapConn.conn); err != nil {
			return err
		}
	}

	return nil
}

func (c *channelPool) Len() int {
	return len(c.getConns())
}

func (c *channelPool) getConns() chan *generalConn {
	c.mu.Lock()
	conns := c.conns
	c.mu.Unlock()
	return conns
}
