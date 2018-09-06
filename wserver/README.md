# wserver

## Usage
### Start Server
1. srv := NewServer(...)
2. srv.ListenAndServe()

```
    addr := ":12345"
	srv := NewServer(addr)
	go func() {
		//time.Sleep(time.Second * time.Duration(5))
		for i := 0; i < 10000; i++ {
			srv.Push(EVENT_NEW_UNIT, fmt.Sprintf("msg %d", i))
			time.Sleep(time.Millisecond*time.Duration(500))
		}
	}()
	srv.ListenAndServe()
```

### Communicate
1. client register message

>client send `{"event":"new_unit"} to server to register"`

2. server push message
```
/// whenever new unit is created,call `push` to push unit data to the registered client
srv.Push("new_unit", "unit json data put here")
```