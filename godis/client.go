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
	id            string
	redisClient   *RedisClient
	mutex         sync.Mutex
	HandleTasks   map[string]*ClientTask
	serviceCache  map[string]*ServiceCache
	maxRetryCount int
	isListening   bool
}



func NewClient(id string, addr string, maxRetryCount int) (*Client, error) {

	redisClient, err := NewRedisClient(addr)
	if err!=nil {
		return nil, err
	}
	key := Consumers(id)
	val := redisClient.pushConn.Keys(key)
	if val.Err()!=nil {
		return nil, val.Err()
	}
	if len(val.Val())!=0 {
		return nil, SCRepeatErr
	}

	key = ConsumerMsgQueen(id)
	redisClient.pushConn.Del(key)

	log.Println("key,%v", val)
	client := Client{
		id: id,
		redisClient: redisClient,
		HandleTasks: make(map[string]*ClientTask),
		serviceCache: make(map[string]*ServiceCache),
		maxRetryCount:maxRetryCount,
	}

	return &client, nil
}



func (c *Client) Call(name string, handlerFunc HandleClientFunc, args ProtoType) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if (!c.isListening) {
		go c.Listen()
		c.isListening = true
	}

	event, err := newEvent(c.id, name, args)
	if err != nil {
		return err
	}

	msg, err := event.packBytes()

	done := make(chan bool)
	c.HandleTasks[event.MsgId] = &ClientTask{
		done:done,
		HandlerFunc:handlerFunc,
	}


	go c.Invoke(event.MsgId, done, string(msg[:]), name, 0)
	go c.IncServiceCount(name)
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
//		log.Println("no cache %v", opt)
		servers := c.redisClient.pushConn.ZRangeByScore(key, opt).Val()
		l := len(servers)
		if l!= 0 {
			server = servers[randServer(l)]
			c.serviceCache[serviceName]= &ServiceCache{//save cache
				id:server,
				expireAt:time.Now().Add(time.Minute).Unix(),
			}
		}else {
			go c.Send404(msgId)
			go c.IncServiceNotFoundCount(serviceName)
			return
		}
	}

	serverKey := ProducerMsgQueen(server)
	c.redisClient.pushConn.RPush(serverKey, string(msg[:]))

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
					if retryCount<c.maxRetryCount {
						go c.Invoke(msgId, done, msg, serviceName, retryCount)
					} else {
						go c.RemoveTimeoutTask(msgId)
						go c.IncRequestTimeoutCount(serviceName)
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
	resp := newResp(msgId, 1001, errMsgMap[1001], nil)
	go c.ProcessFunc(&resp)
}


func (c *Client) Send404(msgId string) {
	log.Println("sorry not found !")
	resp := newResp(msgId, 1000, errMsgMap[1000], nil)
	go c.ProcessFunc(&resp)
}






func (c *Client) Listen() {

	key := ConsumerMsgQueen(c.id)

	for {
		if !c.isListening {
			break;
		}
		msg := c.redisClient.popConn.BLPop(2 * time.Second, key).Val()
		if len(msg)!=0 {
			go c.UnpackMsg(msg[1])
		}

	}
}



func (c *Client) UnpackMsg(msg string) {

	mySlice := []byte(msg)

	resp, err := unPackRespByte(mySlice)
	if err != nil {
		log.Printf("err: %v\n", err)
	}

	go c.ProcessFunc(resp)
}


func (c *Client) ProcessFunc(resp *Resp) {

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

//monitor

func (c *Client) IncServiceCount(serviceName string) {
	key := ServiceCallCount(serviceName)
	c.redisClient.pushConn.Incr(key)
}

func (c *Client) IncRequestTimeoutCount(serviceName string) {
	key := RequestTimeoutCount(serviceName)
	c.redisClient.pushConn.Incr(key)
}

func (c *Client) IncServiceNotFoundCount(serviceName string) {
	key := ServiceNotFoundCount(serviceName)
	c.redisClient.pushConn.Incr(key)
}


