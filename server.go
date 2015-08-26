package main


import (
	"./godis"
)


func main (){

	server, _:=godis.NewServer("4","127.0.0.1:6379")

	h := func(v godis.ProtoType) (int64, godis.ProtoType, error) {
		dd := make(godis.ProtoType)
		dd["r"]=v["a"].(int64)+v["b"].(int64)
		return 0, dd, nil
	}
	server.RegisterTask("hello", &h)
    server.Listen()
}