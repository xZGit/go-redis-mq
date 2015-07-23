package godis


import (
	"sync"
	"log"
	"time"
	"sync/atomic"
)



// Task handler representation
type ClientTask struct {
	done        chan bool
	HandlerFunc HandleClientFunc
}

type Client struct {
	id          string
	redisClient *RedisClient
	mutex       sync.Mutex
	HandleTasks map[string]*ClientTask
	isListening bool
}



func NewClient(id string, host string) (*Client, error) {
	redisClient := NewRedisClient(host)
	key := Consumers()
	val := redisClient.pushConn.SAdd(key, id).Val()

	if (val==0) {
		//		return nil,  errors.New("zerorpc/event interface conversion error")
	}

	client := Client{
		id: id,
		redisClient: redisClient,
		HandleTasks: make(map[string]*ClientTask),
	}

	return &client, nil
}



func (c *Client) Call(name string, handlerFunc HandleClientFunc, args ProtoType, n int) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if (!c.isListening) {
		go c.Listen()
		c.isListening = true
	}

	event, err := newEvent(c.id, name, args)
	if err != nil {
		panic(err)
	}

	msg, err := event.packBytes()

	done := make(chan bool)
	c.HandleTasks[event.MsgId] = &ClientTask{
		done:done,
		HandlerFunc:handlerFunc,
	}

	key := ProducerService(name)
	go c.Invoke(event.MsgId, done, string(msg[:]), key, 0)

	return nil
}

var op int64 = 0
func (c *Client) Invoke(msgId string, done chan bool, msg string, key string, retryCount int) {
	c.mutex.Lock()
	serve := c.redisClient.pushConn.SRandMember(key).Val()
	if serve!="" {
		serveKey := ProducerMsgQueen(serve)
		c.redisClient.pushConn.LPush(serveKey, string(msg[:]))

		atomic.AddInt64(&op, 1)
		log.Println("send: %d", op)
	}else {

	}
	c.mutex.Unlock()
	go func() {
		ttl := time.After(60 * time.Second)
		end := make(chan int64)
		Loop:
		for {
			select {
			case <-done:
				end <- 1
			case <-ttl:
				go func() {
					retryCount=retryCount+1
					log.Println("retry : %d", retryCount)
					if retryCount<MaxRetryCount {
						go c.Invoke(msgId, done, msg, key, retryCount)
					} else {
						go c.RemoveTimeoutTask(msgId)
					}
					end <- 1
				}()
			case <-end:
				break Loop
			}

		}
	}()
	log.Println("finish!")
}


func (c *Client) RemoveTimeoutTask(msgId string) {
	log.Println("sorry timeout!")
	resp := Resp{
		RespInfo:RespInfo{
			Code:1,
			ErrMsg:"request timeout",
		},
	}
	if fn, ok := c.HandleTasks[msgId]; ok {
		(*fn.HandlerFunc)(resp.RespInfo)
		delete(c.HandleTasks, resp.MsgId)
	} else {
		log.Println("fn is Not Found")
	}
}







var o int64 = 0

func (c *Client) Listen() {

	key := ConsumerMsgQueen(c.id)

	for {
		if !c.isListening {
			break;
		}
		msg := c.redisClient.popConn.BLPop(2 * time.Second, key).Val()
		if len(msg)!=0 {
			atomic.AddInt64(&o, 1)
			go c.ProcessFunc(msg[1])
		}

	}
}



func (c *Client) ProcessFunc(msg string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	mySlice := []byte(msg)

	resp, err := unPackRespByte(mySlice)
	if err != nil {
		log.Printf("err: %v\n", err)
	}

	if fn, ok := c.HandleTasks[resp.MsgId]; ok {
		(*fn.HandlerFunc)(resp.RespInfo)
		fn.done <- true
		delete(c.HandleTasks, resp.MsgId)
	} else {
		log.Println("fn Not Found")
	}


	if len(c.HandleTasks)==0 {
		c.isListening = false
	}
}

