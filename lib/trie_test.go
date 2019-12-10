package lib

import (
	"testing"
)

func TestTrie_Find(t *testing.T) {
	trie := NewTrie()
	keyList := []string{"ABCD", "ABC", "AB", "A", "B", "C", "BCD"}

	for _, key := range keyList {
		trie.Insert([]byte(key), key)
	}

	for _, key := range keyList {
		ret, value := trie.Find([]byte(key))
		if ret == false {
			t.Error("find or insert error")
			return
		}
		if value != key {
			t.Error("key value is error")
			return
		}
	}

	if trie.NumberKey != int32(len(keyList)) {
		t.Error("key number is wrong")
	}
}

func TestTrie_Remove(t *testing.T) {
	trie := NewTrie()

	keyList := []string{"ABCD", "ABC", "AB", "A", "B", "C", "BCD"}

	for _, key := range keyList {
		trie.Insert([]byte(key), nil)
	}

	for idx, key := range keyList {
		trie.Remove([]byte(key))

		if trie.NumberKey != int32(len(keyList)-idx-1) {
			t.Error("remove test failed")
			return
		}
		ret, _ := trie.Find([]byte(key))

		if ret {
			t.Error("remove test failed")
			return
		}
	}
}
