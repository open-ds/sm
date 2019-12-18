package lib

type AC struct {
	Trie *Trie
}

func NewAC() *AC {
	trie := NewTrie()
	ac := &AC{Trie: trie}
	ac.Trie.Root.Fail = ac.Trie.Root
	return ac
}

func (ac *AC) Insert(key []byte) {
	ac.Trie.Insert(key, string(key))
}

func (ac *AC) Remove(key []byte) {
	ac.Trie.Remove(key)
}

// 构建AC自动机的时候需要广度优先遍历字典树
func (ac *AC) Build() {
	root := ac.Trie.Root

	ac.Trie.BFS(func(key []byte, node *Node, parent *Node) {
		if node == root {
			return
		}

		// 第一层的节点Fail指针都指向root
		if parent == root {
			node.Fail = root
			return
		}

		ord := key[node.Height]
		next := parent.Fail

		for ; next != root; next = next.Fail {
			if next.Children[ord] != nil {
				break
			}
		}

		if next.Children[ord] != nil {
			node.Fail = next.Children[ord]
		} else {
			node.Fail = root
		}

	})
}

func (ac *AC) Match(key []byte) (position [][]int) {
	root := ac.Trie.Root
	node := root

	for i := 0; i < len(key); i++ {
		ord := key[i]

		for node != root && node.Children[ord] == nil {
			node = node.Fail
		}

		node = node.Children[ord]

		if node == nil {
			node = root
		}

		for current := node; current != root; current = current.Fail {
			if current.IsKey {
				position = append(position, []int{i - current.Height, i})
			}
		}
	}

	return position
}
