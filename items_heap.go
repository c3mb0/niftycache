package niftycache

import "container/heap"

func newItemsHeap() *itemsHeap {
	ih := new(itemsHeap)
	heap.Init(ih)
	return ih
}

type itemsHeap struct {
	items []*item
}

func (ih *itemsHeap) peek() *item {
	if ih.Len() == 0 {
		return nil
	}
	return ih.items[0]
}

func (ih *itemsHeap) update(item *item) {
	heap.Fix(ih, item.index)
}

func (ih *itemsHeap) push(item *item) {
	heap.Push(ih, item)
}

func (ih *itemsHeap) pop() {
	heap.Pop(ih)
}

func (ih *itemsHeap) remove(item *item) {
	heap.Remove(ih, item.index)
}

func (ih itemsHeap) Len() int {
	return len(ih.items)
}

func (ih itemsHeap) Less(i, j int) bool {
	return ih.items[i].expireAt.Before(ih.items[j].expireAt)
}

func (ih itemsHeap) Swap(i, j int) {
	ih.items[i], ih.items[j] = ih.items[j], ih.items[i]
	ih.items[i].index = i
	ih.items[j].index = j
}

func (ih *itemsHeap) Push(x interface{}) {
	item := x.(*item)
	item.index = len(ih.items)
	ih.items = append(ih.items, item)
}

func (ih *itemsHeap) Pop() interface{} {
	old := ih.items
	n := len(old)
	item := old[n-1]
	item.index = -1
	old[n-1] = nil
	ih.items = old[0 : n-1]
	return item
}
