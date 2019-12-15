package lib

import (
	"bufio"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AofWriter struct {
	Filename      string
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
	log.Printf("Load config file: %s\n", filename)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0664)

	if err != nil {
		log.Fatal(err.Error())
	}

	aof := &AofWriter{}
	aof.File = file
	aof.Fsync = 2 // default fsync every second
	aof.Filename = filename
	return aof
}

func (aof *AofWriter) Feed(cmd []byte) {
	log.Println(string(cmd))
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

func (aof *AofWriter) Load(server *Server) {
	log.Printf("AOF Load from file %s\n", aof.Filename)
	reader := bufio.NewReader(aof.File)
	for {
		buf, _, err := reader.ReadLine()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalln(err.Error())
		}

		if buf[0] != 42 {
			log.Fatalln("aof file format error")
		}

		lenArgc, err := strconv.ParseInt(string(buf[1:]), 10, 32)

		if err != nil {
			log.Fatalln(err.Error())
		}
		cmd := make([][]byte, 0)
		for i := 0; i < int(lenArgc); i++ {
			buf, _, err = reader.ReadLine()

			if buf[0] != 36 {
				log.Fatalln("aof file format error")
			}

			lenValue, err := strconv.ParseInt(string(buf[1:]), 10, 32)

			if err != nil {
				log.Fatalln(err.Error())
			}

			value := make([]byte, lenValue+2)
			if lenValue == 0 {
				value = nil
			} else {
				_, err = io.ReadFull(reader, value)
				if err != nil {
					log.Fatalln(err.Error())
				}
			}

			cmd = append(cmd, value[0:lenValue])
		}
		switch len(cmd) {
		case 4:
			server.Insert(string(cmd[1]), cmd[2], cmd[3])
		case 3:
			server.Remove(string(cmd[1]), cmd[2])
		}
	}
}
