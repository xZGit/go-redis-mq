package godis


import (
	"sync"
	"log"
	"time"
	"gopkg.in/redis.v3"
	"strconv"
	"math/rand"
)



// Task handler representation
type ClientTask struct {
	done        chan bool
	HandlerFunc HandleClientFunc
}

type ServiceCache struct {
	id       string
	expireAt int64
}


type Client struct {
	id           string
	redisClient  *RedisClient
	mutex        sync.Mutex
	HandleTasks  map[string]*ClientTask
	serviceCache map[string]*ServiceCache
	isListening  bool
}



func NewClient(id string, host string) (*Client, error) {
	redisClient := NewRedisClient(host)
	key := Consumers(id)
	val := redisClient.pushConn.Keys(key).Val()

	log.Println("key,%v", val)
	client := Client{
		id: id,
		redisClient: redisClient,
		HandleTasks: make(map[string]*ClientTask),
		serviceCache: make(map[string]*ServiceCache),
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


	go c.Invoke(event.MsgId, done, string(msg[:]), name, 0)

	return nil
}




func randServer(l int) int {
	return rand.Intn(l)
}


func (c *Client) Invoke(msgId string, done chan bool, msg string, serviceName string, retryCount int) {
	c.mutex.Lock()

	now := time.Now().Unix()
	var server string
	if sc, ok := c.serviceCache[serviceName]; ok && sc.expireAt > now {
		server = sc.id
	} else {
		key := ProducerService(serviceName)
		opt := redis.ZRangeByScore{
			Min: strconv.FormatInt(now, 10),
			Max: "+inf",
		}
		log.Println("no cache %v", opt)
		servers := c.redisClient.pushConn.ZRangeByScore(key, opt).Val()
		l := len(servers)
		if l!= 0 {
			server = servers[randServer(l)]
			c.serviceCache[serviceName]= &ServiceCache{
				id:server,
				expireAt:time.Now().Add(time.Minute).Unix(),
			}
		}else {
            go c.Send404(msgId)
			return
		}
	}

	serverKey := ProducerMsgQueen(server)
	c.redisClient.pushConn.LPush(serverKey, string(msg[:]))

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
						go c.Invoke(msgId, done, msg, serviceName, retryCount)
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
	//	log.Println("finish!")
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


func (c *Client) Send404(msgId string) {
	log.Println("sorry not found !")
	resp := Resp{
		RespInfo:RespInfo{
			Code:2,
			ErrMsg:"SERVICE NOT FOUND",
		},
	}
	if fn, ok := c.HandleTasks[msgId]; ok {
		(*fn.HandlerFunc)(resp.RespInfo)
		delete(c.HandleTasks, resp.MsgId)
	} else {
		log.Println("fn is Not Found")
	}
}






func (c *Client) Listen() {

	key := ConsumerMsgQueen(c.id)

	for {
		if !c.isListening {
			break;
		}
		msg := c.redisClient.popConn.BLPop(2 * time.Second, key).Val()
		if len(msg)!=0 {
			go c.ProcessFunc(msg[1])
		}

	}
}



func (c *Client) ProcessFunc(msg string) {

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

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(c.HandleTasks)==0 {
		c.isListening = false
	}
}

