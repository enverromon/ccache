package ccache

import (
	"container/list"
	"sync/atomic"
	"time"
)

type ItemState int
const (
	ItemStateNormal ItemState = iota
	ItemStateExpired
	ItemStateUpdating
)

type Sized interface {
	Size() int64
}

type Updatable interface {
	Key() string
}

type TrackedItem interface {
	Value() interface{}
	Release()
	Expired() bool
	TTL() time.Duration
	Expires() time.Time
	Extend(duration time.Duration)
	SetStatus(state ItemState)
}

type nilItem struct{}

func (n *nilItem) Value() interface{} { return nil }
func (n *nilItem) Release()           {}

func (i *nilItem) Expired() bool {
	return true
}

func (i *nilItem) TTL() time.Duration {
	return time.Minute
}

func (i *nilItem) Expires() time.Time {
	return time.Time{}
}

func (i *nilItem) Extend(duration time.Duration) {
}

func (i *nilItem) SetStatus(state ItemState) {
}

var NilTracked = new(nilItem)

type Item struct {
	key        string
	group      string
	promotions int32
	refCount   int32
	expires    int64
	size       int64
	value      interface{}
	element    *list.Element
	state 	   ItemState

	done       chan bool
}

func newItem(key string, value interface{}, expires int64) *Item {
	size := int64(1)
	if sized, ok := value.(Sized); ok {
		size = sized.Size()
	}
	return &Item{
		key:        key,
		value:      value,
		promotions: 0,
		size:       size,
		expires:    expires,
		state:      ItemStateNormal,
		done:	    make(chan bool),
	}
}

func (i *Item) shouldPromote(getsPerPromote int32) bool {
	i.promotions += 1
	return i.promotions == getsPerPromote
}

func (i *Item) Value() interface{} {
	return i.value
}

func (i *Item) track() {
	atomic.AddInt32(&i.refCount, 1)
}

func (i *Item) Release() {
	atomic.AddInt32(&i.refCount, -1)
}

func (i *Item) Expired() bool {
	expires := atomic.LoadInt64(&i.expires)
	return expires < time.Now().Unix()
}

func (i *Item) TTL() time.Duration {
	expires := atomic.LoadInt64(&i.expires)
	return time.Second * time.Duration(expires-time.Now().Unix())
}

func (i *Item) Expires() time.Time {
	expires := atomic.LoadInt64(&i.expires)
	return time.Unix(expires, 0)
}

func (i *Item) Extend(duration time.Duration) {
	atomic.StoreInt64(&i.expires, time.Now().Add(duration).Unix())
}

func (i *Item) SetStatus(state ItemState) {
	i.state = state
}
