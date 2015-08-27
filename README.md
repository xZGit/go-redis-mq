# 基于go 和 redis 的消息服务框架


------



### 1.原理
基于redis的list的rpush和blpop的队列，消费者和服务提供者各有一条消息队列，消费者在服务者的消息队列push一条服务请求，服务提供者得到请求处理后放入消费者消息队列，消费者获得消息并处理。

### 2.优点
1 节点由redis控制，消费者和服务者无需关心消息的来源和去向
2 缓存机制
3 重试机制

### 3.例子
服务提供者 server:


```golang
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
```

消费者client:
```golang
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
	client, _:=godis.NewClient("3", "127.0.0.1:6379", 5)

	var wg sync.WaitGroup
    l:=10000
	for i := 0; i < l; i++ {
		wg.Add(1)
		go Afunction(client,wg)
    }
	wg.Wait()

}
```
monitor：
```golang
package main


import (
	"./godis"
)


func main (){
	m, _:=godis.NewMonitor("127.0.0.1:6379")
	m.Listen()

}
```

### 4 测试
基于2中例子，分别对其进行100，1000，10000次的并发测试
测试机器为OSX Yosemite 10.10.2 ,server,client,redis均在同一主机.

| 次数        | 时间   |   
| --------   | -----:  |
| 100      |   2641ms   |   
| 1000     |   301627ms   |  
| 10000    |   30321700ms  |  