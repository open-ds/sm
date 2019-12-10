package lib

import (
	"sync"
	"sync/atomic"
)

// An implement of trie tree

type Node struct {
	IsKey    bool
	Children map[uint8]*Node
	Height   int
	Value    interface{}
	Lock     sync.Mutex
}

type Trie struct {
	Root       *Node
	NumberNode int32
	NumberKey  int32
}

func NewTrie() *Trie {
	root := &Node{IsKey: false, Children: make(map[uint8]*Node), Height: 0}
	trie := &Trie{Root: root, NumberNode: 0, NumberKey: 0}
	return trie
}

func CreateNode(isKey bool, height int) *Node {
	node := &Node{IsKey: isKey, Height: height, Children: make(map[uint8]*Node)}
	return node
}

func (node *Node) InsertChild(ord uint8, child *Node) {
	node.Children[ord] = child
}

func (node *Node) RemoveChild(ord uint8) {
	delete(node.Children, ord)
}

func (node *Node) GetChild(ord uint8) *Node {
	return node.Children[ord]
}

func (node *Node) Update(isKey bool, value interface{}) (ret bool) {
	node.Lock.Lock()
	defer node.Lock.Unlock()

	ret = !node.IsKey

	node.IsKey = isKey
	node.Value = value

	return ret
}

func (trie *Trie) increaseNumberNode() {
	atomic.AddInt32(&trie.NumberNode, 1)
}

func (trie *Trie) decreaseNumberNode() {
	atomic.AddInt32(&trie.NumberNode, -1)
}

func (trie *Trie) increaseNumberKey() {
	atomic.AddInt32(&trie.NumberKey, 1)
}

func (trie *Trie) decreaseNumberKey() {
	atomic.AddInt32(&trie.NumberKey, -1)
}

func (trie *Trie) Walk(key []byte) (*Node, *Node, int) {
	var i int
	node := trie.Root
	parent := trie.Root

	for i = 0; i < len(key); i++ {
		order := key[i]
		parent = node
		node = node.GetChild(order)

		if node == nil {
			break
		}

	}

	return parent, node, i
}

func (trie *Trie) Insert(key []byte, value interface{}) (oldValue interface{}, ret int) {
	var parent *Node
	var node *Node

	keyLen := len(key)
	ret = 0
	parent = trie.Root
	node = trie.Root

	if trie.Root == nil {
		return oldValue, ret
	}

	for i := 0; i < keyLen; i++ {
		order := key[i]
		parent = node
		parent.Lock.Lock()
		node = node.GetChild(order)

		if node != nil {
			// 最后一个节点是key
			if i == keyLen-1 {
				ret = 1
				oldValue = node.Value
				isKey := node.Update(true, value)
				if isKey {
					trie.increaseNumberKey()
				}
				parent.Lock.Unlock()
				break

			} else { // 不是最后一个节点，释放父节点的锁继续遍历
				parent.Lock.Unlock()
				continue
			}
		} else {
			trie.increaseNumberNode()
			node = CreateNode(i == keyLen-1, i)
			parent.Children[order] = node
			if i == keyLen-1 {
				trie.increaseNumberKey()
			}
			parent.Lock.Unlock()
		}

	}

	return oldValue, ret
}

func (trie *Trie) Remove(key []byte) bool {
	var parent *Node
	var node *Node

	keyLen := len(key)
	parent = trie.Root
	node = trie.Root

	if trie.Root == nil {
		return false
	}

	for i := 0; i < keyLen; i++ {
		order := key[i]
		parent = node
		parent.Lock.Lock()
		node = node.GetChild(order)

		if node != nil {
			if i == keyLen-1 {
				node.Lock.Lock()
				if node.IsKey {
					trie.decreaseNumberKey()
					node.IsKey = false
				}

				if len(node.Children) == 0 {
					trie.decreaseNumberNode()
					parent.RemoveChild(order)
				}

				node.Lock.Unlock()
				parent.Lock.Unlock()
				return false
			}
			parent.Lock.Unlock()
			continue
		} else {
			parent.Lock.Unlock()
			break
		}

	}
	return false
}

func (trie *Trie) Find(key []byte) (ret bool, value interface{}) {
	ret = false
	value = nil
	keyLen := len(key)

	_, node, step := trie.Walk(key)

	if step == keyLen && node.IsKey {
		ret = true
		value = node.Value
	}

	return ret, value
}

func (trie *Trie) SeekAfter(key []byte) (it *Iterator) {
	_, node, _ := trie.Walk(key)

	if node == nil {
		return it
	}

	it = NewIterator(key, node)
	return it
}

func (trie *Trie) SeekBefore(key []byte) []int {
	var i int
	var flags []int
	node := trie.Root

	for i = 0; i < len(key); i++ {
		order := key[i]

		node = node.Children[order]

		if node == nil {
			break
		} else if node.IsKey {
			flags = append(flags, i)
		}

	}

	return flags
}

func (trie *Trie) BFS(fn func(key []byte, node *Node)) {
	queue := NewQueue()
	queue.Put(make([]byte, 0), trie.Root)

	for !queue.Empty() {
		suffix, node := queue.Get()
		fn(suffix, node)

		for ord, child := range node.Children {
			path := make([]byte, len(suffix))
			copy(path, suffix)
			path = append(path, ord)
			queue.Put(path, child)
		}
	}
}
