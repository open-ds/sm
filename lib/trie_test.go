package lib

import (
	"fmt"
	"github.com/open-ds/sm/tests"
	"sync"
	"testing"
)

func TestTrie_Insert(t *testing.T) {
	for i := 0; i < 100; i++ {
		trie := NewTrie()
		var w sync.WaitGroup
		keyChan := make(chan string, 100)
		keyMap := make(map[string]bool)

		for i := 0; i < 100; i++ {
			w.Add(1)
			go func() {
				for {
					key, ok := <-keyChan

					if !ok {
						w.Done()
						break
					}
					trie.Insert([]byte(key), key)
				}
			}()
		}

		for _, term := range tests.GetTerms("../tests/terms.json") {
			keyChan <- term.Ci
			keyMap[term.Ci] = true
		}

		close(keyChan)
		w.Wait()

		if trie.NumberKey != int32(len(keyMap)) {
			t.Error(fmt.Sprintf("trie number key: %d key map: %d", trie.NumberKey, len(keyMap)))
		}

		for _, term := range tests.GetTerms("../tests/terms.json") {
			//fmt.Println(term.Ci)
			ret, value := trie.Find([]byte(term.Ci))
			if !ret || value != term.Ci {
				t.Error(fmt.Sprintf("key %s not found %d", term.Ci, trie.NumberKey))
			}

		}
	}
}
