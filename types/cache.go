package types

type Cache struct {
	*KVMap
	store Store
}

func NewCache(store Store) *Cache {
	c := &Cache{
		store: store,
	}
	c.Reset()
	return c
}

func (c *Cache) Reset() {
	c.KVMap = NewKVMap()
}

func (c *Cache) Set(key []byte, value []byte) {
	c.KVMap.Set(key, value)
}

func (c *Cache) Get(key []byte) (value []byte) {
	value = c.KVMap.Get(key)
	if value != nil {
		return
	}
	value = c.store.Get(key)
	c.Set(key, value)
	return
}

func (c *Cache) Sync() {
	for kvn := c.KVList.head; kvn != nil; kvn = kvn.next {
		c.store.Set(kvn.key, kvn.value)
	}
	c.Reset()
}
