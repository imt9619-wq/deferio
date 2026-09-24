package utils

type TickCache[K any] struct{
	cache      map[uint64]K
	sortedRing []uint64
	ringStart  int
	n          int
}

func NewTickCache[K any](size int) *TickCache[K]{
	if size < 1{
		panic("tickCache size must be >= 1")
	}
	return &TickCache[K]{
		cache: make(map[uint64]K, size),
		sortedRing: make([]uint64, size),
	}
}

func (t *TickCache[K]) Set(tick uint64, data K){
	if _, ok := t.cache[tick]; ok{
		t.cache[tick] = data
		return
	}
	size := len(t.sortedRing)
	if t.n > 0 && tick < t.sortedRing[(t.ringStart+t.n-1)%size]{
		return
	}
	if t.n == size{
		delete(t.cache, t.sortedRing[t.ringStart])
		t.ringStart = (t.ringStart+1)%size
		t.n--
	}
	t.cache[tick] = data
	t.sortedRing[(t.ringStart+t.n)%size] = tick
	t.n++
}

func (t *TickCache[K]) Pop(tick uint64) (data K, ok bool){
	data, ok = t.cache[tick]
	size := len(t.sortedRing)
	dropped := 0
	for i := 0; i < t.n; i++{
		curr := t.sortedRing[(t.ringStart+i)%size]
		if curr > tick{
			break
		}
		delete(t.cache, curr)
		dropped++
	}
	t.ringStart = (t.ringStart+dropped)%size
	t.n -= dropped
	if t.n == 0{
		t.ringStart = 0
	}
	return data, ok
}
