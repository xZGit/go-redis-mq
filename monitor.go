package main


import (
	"./godis"
)


func main (){
	m, _:=godis.NewMonitor("127.0.0.1:6379")
	m.Listen()

}