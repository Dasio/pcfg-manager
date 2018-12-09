package main

import (
	"github.com/dasio/pcfg-manager/cmd"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	//log.SetFormatter(&log.JSONFormatter{})

}
func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
