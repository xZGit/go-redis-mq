package main


import (
	"./godis"
)


func main (){

	exit := make(chan int)


	server, _:=godis.NewServer("4","127.0.0.1")

	h := func(v godis.ProtoType) (godis.ProtoType, error) {
		dd := make(godis.ProtoType)
		dd["r"]=v["a"].(int64)+v["b"].(int64)
		return dd, nil
	}

	c := make(chan int, 2)
	server.RegisterTask("hello", &h, c)


    server.Listen()

	<-c

//    go client.Call("hello")
	<-exit
}