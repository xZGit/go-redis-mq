package main


import (
	"./godis"
    "errors"
	"log"
	"sync/atomic"

)

var ops int64= 0

func Afunction(client *godis.Client, shownum int) {
	h := func(v godis.RespInfo) (interface{}, error){
		atomic.AddInt64(&ops, 1)
     	log.Println("done: %d",ops)
		return nil, errors.New("Ssss")
	}

	dd := make(godis.ProtoType)
	dd["dd"]="fff"

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