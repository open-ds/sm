package lib

import (
	"container/list"
)

type Item struct {
	Key    []byte
	Node   *Node
	Parent *Node
}

type Queue struct {
	List *list.List
}

func NewQueue() *Queue {
	queue := &Queue{}
	queue.List = list.New()
	return queue
}

func (q *Queue) Put(key []byte, node *Node, parent *Node) {
	item := &Item{Key: key, Node: node, Parent: parent}
	q.List.PushBack(item)
}

func (q *Queue) Empty() bool {
	return q.List.Len() == 0
}

func (q *Queue) Get() (key []byte, node *Node, parent *Node) {
	if q.Empty() {
		return key, node, parent
	}

	element := q.List.Front()
	q.List.Remove(element)
	value := element.Value
	item, _ := value.(*Item)
	return item.Key, item.Node, item.Parent
}

type Stack struct {
	List *list.List
}

func NewStack() (stack *Stack) {
	stack = &Stack{}
	stack.List = list.New()
	return stack
}

func (s *Stack) Empty() bool {
	return s.List.Len() == 0
}

func (s *Stack) Pop() (key []byte, node *Node) {
	if s.Empty() {
		return key, node
	}

	ele := s.List.Back()
	s.List.Remove(ele)
	value := ele.Value
	item, _ := value.(*Item)
	return item.Key, item.Node
}

func (s *Stack) Push(key []byte, node *Node) {
	s.List.PushBack(&Item{Key: key, Node: node})
}

type Iterator struct {
	Queue *Queue
}

func NewIterator(key []byte, node *Node, parent *Node) (it *Iterator) {
	it = &Iterator{Queue: NewQueue()}
	it.Queue.Put(key, node, parent)
	return it
}

func (it *Iterator) HasNext() bool {
	return it != nil && !it.Queue.Empty()
}

func (it *Iterator) Next() (key []byte, node *Node, parent *Node) {
	key, node, parent = it.Queue.Get()

	for ord, child := range node.Children {
		suffix := make([]byte, len(key))
		copy(suffix, key)
		suffix = append(suffix, ord)
		it.Queue.Put(suffix, child, node)
	}

	return key, node, parent
}
