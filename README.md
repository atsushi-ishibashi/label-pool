# label-pool
Inspired by [fatih/pool](https://github.com/fatih/pool)

```
pool, err := NewChannelPool(100, func() (net.Conn, error) { return net.Dial("tcp", addr) })

conn, err := pool.Get("1111")

if _, err := conn.Write([]byte("message")); err != nil {
  conn.Close()
}
```

### Concerned
- around `setLabelConn`
