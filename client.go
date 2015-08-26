package main


import (
	"./godis"
    "errors"
	"log"
	"sync"
	"sync/atomic"
	"syscall"
)

var ops int64= 0

func Afunction(client *godis.Client, wg sync.WaitGroup) {
	var ts, te syscall.Timeval
	syscall.Gettimeofday(&ts)
	start:=(int64(ts.Sec)*1e3 + int64(ts.Usec)/1e3)

	h := func(v godis.RespInfo) (interface{}, error){
		defer wg.Add(-1)
		syscall.Gettimeofday(&te)
		end:=(int64(te.Sec)*1e3 + int64(te.Usec)/1e3)
		cost:=end-start
		atomic.AddInt64(&ops, cost)
		log.Println("cost: %d",ops)
		return nil, errors.New("Ssss")
	}

	n := make(godis.ProtoType)
	n["a"]=1
	n["b"]=2
	client.Call("hello",&h,n)

}


func main (){
	client, _:=godis.NewClient("3", "127.0.0.1:6379",5)

	var wg sync.WaitGroup
    l:=10000
	for i := 0; i < l; i++ {
		wg.Add(1)
		go Afunction(client,wg)
    }
	wg.Wait()

}