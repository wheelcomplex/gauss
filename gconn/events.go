package gconn

import (
	"bytes"
	_ "fmt"

	"github.com/zond/gotomic"
)

type Key []byte

type Listener struct {
	HashTable *gotomic.Hash
}

func (self Key) HashCode() uint32 {
	var rval uint32
	for c := range self {
		rval = rval + uint32(c)
	}
	return rval
}

func (self Key) Equals(t gotomic.Thing) bool {
	return bytes.Equal(self, t.(Key))
}

func (self *Conn) RegisterListener() *Listener {
	h := gotomic.NewHash()
	return &Listener{
		HashTable: h,
	}
}

func (self *Listener) Put(k, t gotomic.Thing) {
	self.HashTable.Put(Key(k.(Key)), t)
}

func (self *Listener) Get(k gotomic.Thing) (gotomic.Thing, bool) {
	return self.HashTable.Get(Key(k.(Key)))
}
