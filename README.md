# niftycache

```go
cb := func(k string, v interface{}) {
	fmt.Printf("key: %s | value: %+v\n", k, v)
}

cache := niftycache.New(10*time.Minute,
	niftycache.ExpireCallback(cb),
	niftycache.RemoveCallback(cb),
	niftycache.SetCallback(cb),
	niftycache.UpdateCallback(cb),
	// controls whether a successful cache hit extends the item's ttl (default false)
	niftycache.ExtendTTLOnHit(),
	// controls the max amount of items that can be expired at once (default 10000)
	niftycache.MaxExpires(10000),
	// controls the max amount of concurrent callbacks that can be fired (default 1000)
	niftycache.MaxCallbacks(1000),
)
```