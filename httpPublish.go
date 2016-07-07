package main

import (
	"fmt"
	"time"
	"net/http"
	"encoding/json"
)

type Person struct {
	Name	string	`json:"name"`
	Sex	string	`json:"sex"`
	Age	int	`json:"age"`
}

func handleDebugVars(w http.ResponseWriter, r *http.Request) {
	var p *Person = &Person{
		Name :	"wangchunyan",
		Sex :	"man",
		Age :	26,
	}
	str, err := json.Marshal(&p)
	if err != nil {
		return
	}
	fmt.Fprintf(w, string(str))
	return
}

func unpackPerson() error {
	var p Person
	str := `{"name":"wangchunyan","sex":"man","age":26}`
	json.Unmarshal([]byte(str), &p)
	fmt.Printf("p.name is:%v\n", p.Name)
	return nil
}

func main() {
	http.HandleFunc("/debug/vars", handleDebugVars)

	http.ListenAndServe("127.0.0.1:8088", nil)
	for {
		time.Sleep(time.Second)
	}
}

