package main


import (
	"./godis"
    "errors"
	"log"
	"sync/atomic"
//	"time"
	"syscall"

)

var ops int64= 0

func Afunction(client *godis.Client, shownum int) {
	var ts, te syscall.Timeval
	syscall.Gettimeofday(&ts)
	start:=(int64(ts.Sec)*1e3 + int64(ts.Usec)/1e3)

	h := func(v godis.RespInfo) (interface{}, error){
		log.Println("v: %v",v.Data["r"])
		syscall.Gettimeofday(&te)
		end:=(int64(te.Sec)*1e3 + int64(te.Usec)/1e3)
		cost:=end-start
		atomic.AddInt64(&ops, cost)
		log.Println("cost: %d",ops)
		return nil, errors.New("Ssss")
	}

	dd := make(godis.ProtoType)
	dd["a"]=1
	dd["b"]=2
	client.Call("hello",&h,dd,shownum)

}




func main (){
	c := make(chan int)
	client, _:=godis.NewClient("3", "127.0.0.1")
	for i := 0; i < 10000; i++ {
			go Afunction(client,i)
		}


	log.Println("finish!!!!",<-c)
}