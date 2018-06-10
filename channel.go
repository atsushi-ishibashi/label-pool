package labelpool

import (
	"io"
	"net"
	"sync"
	"time"
)

// channelPool implements the Pool interface based on buffered channels.
type channelPool struct {
	cmLock  sync.RWMutex
	connMap map[string]net.Conn

	// net.Conn generator
	factory Factory
}

// Factory is a function to create new connections.
type Factory func() (net.Conn, error)

func NewChannelPool(factory Factory) (Pool, error) {
	c := &channelPool{
		connMap: make(map[string]net.Conn),
		factory: factory,
	}

	go c.loopCheck()

	return c, nil
}

func (c *channelPool) loopCheck() {
	for {
		c.cmLock.Lock()
		for label, conn := range c.connMap {
			one := make([]byte, 1)
			conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			if _, err := conn.Read(one); err == io.EOF {
				delete(c.connMap, label)
				conn.Close()
			}
		}
		c.cmLock.Unlock()

		time.Sleep(60 * time.Second)
	}
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

// Get implements the Pool interfaces Get() method. If there is no new
// connection available in the pool, a new connection will be created via the
// Factory() method.
func (c *channelPool) Get(label string) (net.Conn, error) {
	if conn, ok := c.getLabelConn(label); ok {
		return conn, nil
	}

	conn, err := c.factory()
	if err != nil {
		return nil, err
	}
	c.setLabelConn(label, conn)
	return conn, nil
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

func (c *channelPool) LenMap() int {
	c.cmLock.RLock()
	defer c.cmLock.RUnlock()
	return len(c.connMap)
}
