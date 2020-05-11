package labelpool

import "net"

type PoolConn struct {
	net.Conn
	c *channelPool
}

// Close() puts the given connects back to the pool instead of closing it.
func (p *PoolConn) Close() error {
	p.c.deleteLabelConn(p.Conn)
	if p.Conn != nil {
		return p.Conn.Close()
	}
	return nil
}

// newConn wraps a standard net.Conn to a poolConn net.Conn.
func (c *channelPool) wrapConn(conn net.Conn) net.Conn {
	return &PoolConn{c: c, Conn: conn}
}
