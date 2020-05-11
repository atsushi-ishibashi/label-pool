package labelpool

import (
	"errors"
	"net"
	"sync"
)

// channelPool implements the Pool interface based on buffered channels.
type channelPool struct {
	cmLock  sync.RWMutex
	connMap map[int]net.Conn

	// net.Conn generator
	factory Factory

	mod int
}

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

func NewChannelPool(mod int, factory Factory) (Pool, error) {
	if mod <= 0 {
		return nil, errors.New("invalid mod")
	}
	return &channelPool{
		connMap: make(map[int]net.Conn),
		factory: factory,
		mod:     mod,
	}, nil
}

func (c *channelPool) setLabelConn(label string, conn net.Conn) {
	c.cmLock.Lock()
	defer c.cmLock.Unlock()
	m := c.labeltoi(label)
	if c, ok := c.connMap[m]; ok {
		c.Close()
	}
	c.connMap[m] = conn
}

func (c *channelPool) getLabelConn(label string) (conn net.Conn, ok bool) {
	c.cmLock.RLock()
	defer c.cmLock.RUnlock()
	conn, ok = c.connMap[c.labeltoi(label)]
	return
}

func (c *channelPool) labeltoi(label string) int {
	total := 0
	for _, v := range []byte(label) {
		total += int(v)
	}
	return total % c.mod
}

// Get implements the Pool interfaces Get() method. If there is no new
// connection available in the pool, a new connection will be created via the
// Factory() method.
func (c *channelPool) Get(label string) (net.Conn, error) {
	if conn, ok := c.getLabelConn(label); ok {
		return c.wrapConn(conn), nil
	}

	conn, err := c.factory()
	if err != nil {
		return nil, err
	}
	c.setLabelConn(label, conn)
	return c.wrapConn(conn), nil
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
