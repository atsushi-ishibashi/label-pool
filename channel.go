package labelpool

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// channelPool implements the Pool interface based on buffered channels.
type channelPool struct {
	// storage for our net.Conn connections
	mu    sync.RWMutex
	conns chan net.Conn

	cmLock  sync.RWMutex
	connMap map[string]net.Conn

	// net.Conn generator
	factory Factory
}

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

// NewChannelPool returns a new pool based on buffered channels with an initial
// capacity and maximum capacity. Factory is used when initial capacity is
// greater than zero to fill the pool. A zero initialCap doesn't fill the Pool
// until a new Get() is called. During a Get(), If there is no new connection
// available in the pool, a new connection will be created via the Factory()
// method.
func NewChannelPool(initialCap, maxCap int, factory Factory) (Pool, error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	c := &channelPool{
		conns:   make(chan net.Conn, maxCap),
		connMap: make(map[string]net.Conn),
		factory: factory,
	}

	// create initial connections, if something goes wrong,
	// just close the pool error out.
	for i := 0; i < initialCap; i++ {
		conn, err := factory()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.conns <- conn
	}

	return c, nil
}

func (c *channelPool) setLabelConn(label string, conn net.Conn) {
	c.cmLock.Lock()
	defer c.cmLock.Unlock()
	c.connMap[label] = conn
}

func (c *channelPool) getLabelConn(label string) (conn net.Conn, ok bool) {
	c.cmLock.RLock()
	defer c.cmLock.RUnlock()
	conn, ok = c.connMap[label]
	return
}

func (c *channelPool) getConnsAndFactory() (chan net.Conn, Factory) {
	c.mu.RLock()
	conns := c.conns
	factory := c.factory
	c.mu.RUnlock()
	return conns, factory
}

// Get implements the Pool interfaces Get() method. If there is no new
// connection available in the pool, a new connection will be created via the
// Factory() method.
func (c *channelPool) Get(label string) (net.Conn, error) {
	if conn, ok := c.getLabelConn(label); ok {
		return c.wrapConn(conn), nil
	}

	conns, factory := c.getConnsAndFactory()
	if conns == nil {
		return nil, ErrClosed
	}

	// wrap our connections with out custom net.Conn implementation (wrapConn
	// method) that puts the connection back to the pool if it's closed.
	select {
	case conn := <-conns:
		if conn == nil {
			return nil, ErrClosed
		}
		wconn := c.wrapConn(conn)
		c.setLabelConn(label, wconn)
		return wconn, nil
	default:
		conn, err := factory()
		if err != nil {
			return nil, err
		}
		wconn := c.wrapConn(conn)
		c.setLabelConn(label, wconn)
		return wconn, nil
	}
}

// put puts the connection back to the pool. If the pool is full or closed,
// conn is simply closed. A nil conn will be rejected.
func (c *channelPool) put(conn net.Conn) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conns == nil {
		// pool is closed, close passed connection
		return conn.Close()
	}

	// put the resource back into the pool. If the pool is full, this will
	// block and the default case will be executed.
	select {
	case c.conns <- conn:
		return nil
	default:
		// pool is full, close passed connection
		c.deleteLabelConn(conn)
		return conn.Close()
	}
}

func (c *channelPool) deleteLabelConn(conn net.Conn) {
	c.cmLock.Lock()
	defer c.cmLock.Unlock()
	for k, v := range c.connMap {
		if v == conn {
			delete(c.connMap, k)
		}
	}
}

func (c *channelPool) Close() {
	c.mu.Lock()
	conns := c.conns
	c.conns = nil
	c.factory = nil
	c.mu.Unlock()

	if conns == nil {
		return
	}

	close(conns)
	for conn := range conns {
		conn.Close()
	}
}

func (c *channelPool) Len() int {
	conns, _ := c.getConnsAndFactory()
	return len(conns)
}

func (c *channelPool) LenMap() int {
	return len(c.connMap)
}
