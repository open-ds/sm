package tests

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Term struct {
	Ci          string `json:"ci"`
	Explanation string `json:"explanation"`
}

func GetTerms(filename string) []Term {
	var terms []Term
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln(err.Error())
	}

	err = json.Unmarshal(buf, &terms)
	if err != nil {
		log.Fatalln(err.Error())
	}

	return terms
}
