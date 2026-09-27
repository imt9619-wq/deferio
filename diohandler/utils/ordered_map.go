package utils

import "cmp"

type OrderedMap[T cmp.Ordered, K any] struct {
	unorderedMap map[T]K
	sortedRing   []T
	ringStart    int
	n            int
}

func NewOrderedCache[T cmp.Ordered, K any](size int) *OrderedMap[T, K] {
	if size < 1 {
		panic("OrderedCache size must be >= 1")
	}
	return &OrderedMap[T, K]{
		unorderedMap:      make(map[T]K, size),
		sortedRing: make([]T, size),
	}
}

func (t *OrderedMap[T, K]) Set(key T, data K){
	if _, ok := t.unorderedMap[key]; ok{
		t.unorderedMap[key] = data
		return
	}
	size := len(t.sortedRing)
	if t.n == size{
		delete(t.unorderedMap, t.sortedRing[t.ringStart])
		t.ringStart = (t.ringStart + 1) % size
		t.n--
	}
	t.unorderedMap[key] = data
	i := t.ringStart+t.n
	t.sortedRing[i%size] = key
	if t.n > 0{
		for t.sortedRing[i%size] < t.sortedRing[(i-1)%size] && (i-1)%size != t.ringStart{
			t.sortedRing[i%size], t.sortedRing[(i-1)%size] = t.sortedRing[(i-1)%size], t.sortedRing[i%size]
			i += size-1
		}
	}
	t.n++
}

func (t *OrderedMap[T, K]) Get(key T) (K, bool){
	data, ok := t.unorderedMap[key]
	return data, ok
}
