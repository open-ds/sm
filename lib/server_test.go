package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func RandomASCIIString() string {
	bytes := make([]byte, 0)
	r := rand.Intn(126) + 1

	for i := 0; i < r; i++ {
		bytes = append(bytes, uint8(rand.Intn(26)+65))
	}

	return string(bytes)
}

func CreateTrie(name string) error {
	url := "http://localhost:8080/api/trie"
	contentType := "application/json"
	resp, err := http.Post(url, contentType, strings.NewReader(fmt.Sprintf(`{"name":"%s"}`, name)))

	if err != nil {
		return err
	}

	fmt.Println(resp.StatusCode)
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(body)
		return errors.New(fmt.Sprintf(`%d %s`, resp.StatusCode, string(body)))
	}

	return nil
}

func InsertKey(c *http.Client, key string) error {
	url := "http://localhost:8080/api/trie/test"
	contentType := "application/json"
	resp, err := c.Post(url, contentType, strings.NewReader(fmt.Sprintf(`["%s"]`, key)))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf(`%d %s %s`, resp.StatusCode, string(body), key))
	}
	_, err = io.Copy(ioutil.Discard, resp.Body)

	resp.Body.Close()

	return nil
}

func GetKey(key string) (string, error) {
	url := fmt.Sprintf("http://localhost:8080/api/test/%s", key)
	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf(`%d %s`, resp.StatusCode, string(body)))
	}

	return string(body), nil
}

type Config struct {
	Addr string `yaml:"addr"`
	AOF  struct {
		Fsync    int    `yaml:"fsync"`
		FileName string `yaml:"filename"`
	}
}

func TestServer_Insert(t *testing.T) {
	//err := os.Remove("./aof.log")
	//if err != nil {
	//	t.Error(err.Error())
	//	return
	//}

	wg := sync.WaitGroup{}
	err := CreateTrie("test")
	keyMap := make(map[string]bool)
	if err != nil {
		t.Error(err.Error())
	}
	keyChan := make(chan string, 100)
	okChan := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for {
				key, ok := <-keyChan
				if !ok {
					break
				}
				c := &http.Client{
					Timeout: time.Second,
				}
				err := InsertKey(c, key)
				if err != nil {
					t.Error(err.Error())
					continue
				}
				okChan <- key
			}
			wg.Done()
		}()
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for key := range okChan {
				fmt.Println(key)
				_, err := GetKey(key)
				if err != nil {
					t.Error(err)
					continue
				}
			}
			wg.Done()
		}()

	}

	for i := 0; i < 100; i++ {
		s := RandomASCIIString()
		keyMap[s] = true
		keyChan <- s
	}

	close(keyChan)

	wg.Wait()
}

type Term struct {
	Ci          string `json:"ci"`
	Explanation string `json:"explanation"`
}

func TestServer_HandleKeyInsert(t *testing.T) {
	err := CreateTrie("test")

	if err != nil {
		t.Error(err.Error())
	}

	keyChan := make(chan string, 100)
	okChan := make(chan string, 300000)
	wg := sync.WaitGroup{}
	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func() {
			c := &http.Client{
				Timeout: time.Second * 3,
			}
			for {
				key, ok := <-keyChan
				if !ok {
					break
				}
				err := InsertKey(c, key)
				if err != nil {
					t.Error(err.Error())
					continue
				}
				okChan <- key
			}
			wg.Done()
		}()
	}
	//
	var terms []Term
	terms = make([]Term, 0)

	buf, err := ioutil.ReadFile("ci.json")

	if err != nil {
		t.Error(err.Error())
	}

	err = json.Unmarshal(buf, &terms)
	for _, term := range terms {
		keyChan <- term.Ci
	}

	close(keyChan)
	wg.Wait()
	close(okChan)

	var okTerms []byte
	okTerms = make([]byte, 0)

	for {
		key, ok := <-okChan
		if !ok {
			break
		}
		okTerms = append(okTerms, []byte(key+"\n")...)
	}
	err = ioutil.WriteFile("insert.log", okTerms, 0664)

	if err != nil {
		t.Error(err.Error())
	}

}

func TestServer_CreateTrie(t *testing.T) {
	var terms []Term
	terms = make([]Term, 0)

	buf, err := ioutil.ReadFile("ci.json")

	if err != nil {
		t.Error(err.Error())
	}

	err = json.Unmarshal(buf, &terms)
	keyMap := make(map[string]string)
	for _, term := range terms {
		keyMap[term.Ci] = term.Ci
	}

	fmt.Println(len(keyMap))
}
