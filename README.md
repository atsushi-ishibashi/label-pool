# label-pool
Inspired by [fatih/pool](https://github.com/fatih/pool)

```
pool, err := NewChannelPool(100, func() (net.Conn, error) { return net.Dial("tcp", addr) })

conn, err := pool.Get("1111")

if _, err := conn.Write([]byte("message")); err != nil {
  conn.Close()
}
// error出ないときはClose()しなくてよい
```

### Concerned
- `setLabelConn`が並列で呼ばれた場合に上書きしちゃうのでは?
