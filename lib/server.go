package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Server struct {
	DB     map[string]*Trie
	AOF    *AofWriter
	Config struct {
		Addr string `yaml:"addr"`
		AOF  struct {
			Fsync    int    `yaml:"fsync"`
			FileName string `yaml:"filename"`
		}
		Debug bool
	}
	WG    sync.WaitGroup
	Mutex sync.Mutex
}

type SearchRequest struct {
	Name   string   `json:"name"`
	Key    []string `json:"key"`
	Option string   `json:"option"`
	Limit  int      `json:"limit"`
}

func (server *Server) CreateTrie(name string) {
	fmt.Println(name)
	server.Mutex.Lock()
	server.DB[name] = NewTrie()
	server.Mutex.Unlock()
}

func (server *Server) GetTrie(name string) *Trie {
	server.Mutex.Lock()
	trie, ok := server.DB[name]
	server.Mutex.Unlock()
	if !ok {
		return nil
	}
	return trie
}

func (server *Server) Insert(name string, key []byte, value interface{}) error {
	trie, ok := server.DB[name]
	if !ok {
		return errors.New(fmt.Sprintf("trie name `%s` not found", name))
	}
	trie.Insert(key, value)
	server.AOF.Feed(ConvertInsert(name, string(key), ""))
	return nil
}

func (server *Server) Remove(name string, key string) error {
	trie, ok := server.DB[name]
	if !ok {
		return errors.New(fmt.Sprintf("trie name `%s` not found", name))
	}
	trie.Remove([]byte(key))
	server.AOF.Feed(ConvertRemove(name, key))
	return nil
}

func (server *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	var searchRequest SearchRequest
	var searchResponse map[string][]string

	searchResponse = make(map[string][]string)

	if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if searchRequest.Name == "" {
		http.Error(w, "name is required", 400)
		return
	}

	if searchRequest.Limit == 0 {
		searchRequest.Limit = 10
	}

	trie := server.GetTrie(searchRequest.Name)

	if trie == nil {
		http.Error(w, "no trie found", 400)
		return
	}

	for _, key := range searchRequest.Key {
		searchResponse[key] = make([]string, 0)

		switch searchRequest.Option {
		case "forward":
			flags := trie.SeekBefore([]byte(key))
			for _, idx := range flags {
				searchResponse[key] = append(searchResponse[key], string(key[0:idx]))
			}
		case "backward":
			it := trie.SeekAfter([]byte(key))
			count := 0
			for it.HasNext() && count < searchRequest.Limit {
				k, node := it.Next()
				if node.IsKey {
					searchResponse[key] = append(searchResponse[key], string(k))
				}
			}
		}

	}

	if err := json.NewEncoder(w).Encode(searchResponse); err != nil {
		http.Error(w, err.Error(), 500)
	}

}

func (server *Server) HandleKeyInsert(w http.ResponseWriter, r *http.Request) {
	var postData []string

	if err := json.NewDecoder(r.Body).Decode(&postData); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	params := mux.Vars(r)
	name := params["name"]

	trie := server.GetTrie(name)

	if trie == nil {
		http.Error(w, "no trie found", 400)
		return
	}

	for _, key := range postData {
		if err := server.Insert(name, []byte(key), nil); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	if err := json.NewEncoder(w).Encode(make(map[string]interface{})); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (server *Server) HandleKeyRemove(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	name := params["name"]
	key := params["key"]

	if err := server.Remove(name, key); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := json.NewEncoder(w).Encode(make(map[string]interface{})); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

type KeyGetResponse struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (server *Server) HandleKeyGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	name := params["name"]
	key := params["key"]

	_, ok := server.DB[name]

	if !ok {
		http.Error(w, "trie not found", 404)
		return
	}

	ret, value := server.DB[name].Find([]byte(key))

	if !ret {
		http.Error(w, "key not found", 404)
		return
	}

	var response KeyGetResponse
	response = KeyGetResponse{
		Key:   key,
		Value: value,
	}

	if err := json.NewEncoder(w).Encode(&response); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (server *Server) HandleTrieCreate(w http.ResponseWriter, r *http.Request) {
	var postData map[string]string
	var err error

	if err = json.NewDecoder(r.Body).Decode(&postData); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	name, ok := postData["name"]

	if !ok {
		http.Error(w, "name must be specified", 400)
		return
	}

	trie := server.GetTrie(name)

	if trie == nil {
		server.CreateTrie(name)
	}

	if err := json.NewEncoder(w).Encode(make(map[string]interface{})); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

type TrieStateResponse struct {
	Name       string `json:"name"`
	NumberNode int32  `json:"number_node"`
	NumberKey  int32  `json:"number_key"`
}

func (server *Server) HandleTrieState(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	name := params["name"]

	trie := server.GetTrie(name)
	if trie == nil {
		http.Error(w, fmt.Sprintf("trie `%s` not found", name), 404)
		return
	}

	var resp TrieStateResponse
	resp = TrieStateResponse{
		Name:       name,
		NumberNode: trie.NumberNode,
		NumberKey:  trie.NumberKey,
	}
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

}

func (server *Server) InitHTTPServer() {

	r := mux.NewRouter()
	r.HandleFunc("/api/trie/search", server.HandleSearch).Methods(http.MethodPost)
	r.HandleFunc("/api/trie", server.HandleTrieCreate).Methods(http.MethodPost)
	r.HandleFunc("/api/trie/{name}", server.HandleTrieState).Methods(http.MethodGet)
	r.HandleFunc("/api/trie/{name}", server.HandleKeyInsert).Methods(http.MethodPost)
	r.HandleFunc("/api/trie/{name}/{key}", server.HandleKeyRemove).Methods(http.MethodDelete)
	r.HandleFunc("/api/trie/{name}/{key}", server.HandleKeyGet).Methods(http.MethodGet)

	// debug模式打开pprof
	if server.Config.Debug {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

		r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		r.Handle("/debug/pprof/block", pprof.Handler("block"))
		r.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		r.Handle("/debug/pprof/profile", pprof.Handler("profile"))
	}

	go func() {
		log.Printf("Init HTTP Server addr: %s\n", server.Config.Addr)
		if err := http.ListenAndServe(server.Config.Addr, r); err != nil {
			log.Fatal(err.Error())
		}
	}()
}

func (server *Server) InitAOF() {
	if server.Config.AOF.Fsync == 2 {
		server.AOF.Cron()
	}
}

func NewServer() *Server {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	server := &Server{}
	server.DB = make(map[string]*Trie)
	// default aof is disabled
	server.Config.AOF.Fsync = -1
	server.Config.AOF.FileName = "./aof.log"

	return server
}

func (server *Server) InitConfig(configFile string) {
	log.Printf("Load config file: %s\n", configFile)
	var err error
	var buf []byte

	buf, err = ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if err = yaml.Unmarshal(buf, &server.Config); err != nil {
		log.Fatalln(err.Error())
	}
	if server.Config.AOF.Fsync != -1 {
		server.AOF = NewAOF(server.Config.AOF.FileName)
	}

}

func (server *Server) Serve() {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	server.InitHTTPServer()
	server.InitAOF()
	<-signals
	if server.Config.AOF.Fsync != -1 {
		server.AOF.Close()
	}
}
