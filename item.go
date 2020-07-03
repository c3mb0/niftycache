package niftycache

import "time"

type item struct {
	key      string
	value    interface{}
	ttl      time.Duration
	expireAt time.Time
	index    int
}

func newItem(key string, value interface{}, ttl time.Duration) *item {
	item := &item{
		value: value,
		ttl:   ttl,
		key:   key,
	}
	item.touch()
	return item
}

func (item *item) update(value interface{}) {
	item.value = value
	item.touch()
}

func (item *item) touch() {
	item.expireAt = time.Now().Add(item.ttl)
}

func (item *item) expired() bool {
	return item.expireAt.Before(time.Now())
}
