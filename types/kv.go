package types

import . "github.com/zballs/comit/util"

type KVNode struct {
	key        []byte
	value      []byte
	next, prev *KVNode
}

type KVList struct {
	head, tail *KVNode
}

func NewKVList() *KVList {
	return &KVList{}
}

func (kvl *KVList) Push(key []byte, value []byte) *KVNode {
	kvn := &KVNode{
		key:   key,
		value: value,
	}
	if kvl.head == nil {
		kvl.head = kvn
	} else {
		kvl.tail.next = kvn
		kvn.prev = kvl.tail
	}
	kvl.tail = kvn
	return kvn
}

func (kvl *KVList) Update(value []byte, kvn *KVNode) {
	kvn.value = value
	if kvl.tail != kvn {
		if kvl.head == kvn {
			kvn.next.prev = nil
			kvl.head = kvn.next
		} else {
			kvn.prev.next = kvn.next
			kvn.next.prev = kvn.prev
		}
		kvl.tail.next = kvn
		kvn.prev = kvl.tail
		kvl.tail = kvn
		kvn.next = nil
	}
}

type KVMap struct {
	*KVList
	m map[string]*KVNode
}

func NewKVMap() *KVMap {
	return &KVMap{
		NewKVList(),
		make(map[string]*KVNode),
	}
}

func (kvm *KVMap) Set(key []byte, value []byte) {
	keystr := BytesToHexstr(key)
	kvn := kvm.m[keystr]
	if kvn == nil {
		kvm.m[keystr] = kvm.KVList.Push(key, value)
	} else {
		kvm.KVList.Update(value, kvn)
	}
}

func (kvm *KVMap) Get(key []byte) []byte {
	keystr := BytesToHexstr(key)
	kvn := kvm.m[keystr]
	if kvn != nil {
		return kvn.value
	}
	return nil
}
