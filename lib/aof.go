package lib

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AofWriter struct {
	Buffer        []byte
	Mutex         sync.RWMutex
	SyncOffset    int32
	CurrentOffset int32
	File          *os.File
	Fsync         int
	Ticker        *time.Ticker
}

func LogIt(msg string) {
	log.Println(msg)
}

func ConvertInsert(name string, key string, value string) []byte {
	op := "INSERT"
	params := []string{
		"*4",
		"$" + strconv.Itoa(len(op)),
		op,
		"$" + strconv.Itoa(len(name)),
		name,
		"$" + strconv.Itoa(len(key)),
		key,
		"$" + strconv.Itoa(len(value)),
		value,
	}
	cmd := strings.Join(params, "\r\n")
	return []byte(cmd)
}

func ConvertRemove(name string, key string) []byte {
	op := "REMOVE"

	params := []string{
		"*3",
		"$" + strconv.Itoa(len(op)),
		op,
		"$" + strconv.Itoa(len(name)),
		name,
		"$" + strconv.Itoa(len(key)),
		key,
	}
	cmd := strings.Join(params, "\r\n")
	return []byte(cmd)
}

func NewAOF(filename string) *AofWriter {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0664)

	if err != nil {
		log.Fatal(err.Error())
	}

	aof := &AofWriter{}
	aof.File = file
	aof.Fsync = 2 // default fsync every second
	return aof
}

func (aof *AofWriter) Feed(cmd []byte) {
	aof.Mutex.Lock()
	aof.Buffer = append(aof.Buffer, cmd...)
	aof.CurrentOffset += int32(len(cmd))
	aof.Mutex.Unlock()
}

// Write buffer to disk
func (aof *AofWriter) Flush() {
	aof.Mutex.RLock()
	n, err := aof.File.Write(aof.Buffer)
	aof.Mutex.RUnlock()

	if err != nil {
		// log it
		LogIt(err.Error())
		return
	}

	aof.Mutex.Lock()
	aof.Buffer = aof.Buffer[n:]
	aof.SyncOffset = int32(n)
	aof.Mutex.Unlock()
}

func (aof *AofWriter) Sync() {
	err := aof.File.Sync()
	if err != nil {
		//log it
		LogIt(err.Error())
	}
}

func (aof *AofWriter) Close() {
	if aof.Ticker != nil {
		aof.Ticker.Stop()
	}

	aof.Flush()
	aof.Sync()
	err := aof.File.Close()
	if err != nil {
		//log it
		LogIt(err.Error())
	}
}

func (aof *AofWriter) Cron() {
	if aof.Fsync == 2 {
		aof.Ticker = time.NewTicker(time.Second)
		go func() {
			for {
				<-aof.Ticker.C
				aof.Flush()
				aof.Sync()
			}
		}()
	}
}
