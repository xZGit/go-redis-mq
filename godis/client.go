package godis


import(
	"sync"
    "log"
	"time"
	"sync/atomic"
)



// Task handler representation
type ClientTask struct {
	TaskId    string
	HandlerFunc HandleClientFunc
}

type Client struct {
	id string
	redisClient   *RedisClient
	mutex         sync.Mutex
	HandleTasks [] *ClientTask
	isListening      bool
}



func NewClient(id string, host string) (*Client, error) {
	redisClient := NewRedisClient(host)
	key := Consumers()
	val := redisClient.conn.SAdd(key, id).Val()

	if(val==0){
//		return nil,  errors.New("zerorpc/event interface conversion error")
	}

	client:=Client{
		id:id,
		redisClient:redisClient,
		HandleTasks :make([]*ClientTask, 0),
	}

	return &client,nil
}


var op int64= 0
func (c *Client) Call(name string, handlerFunc HandleClientFunc, args ProtoType,n int)  error {
    c.mutex.Lock()
	defer c.mutex.Unlock()

	if(!c.isListening){
		go c.Listen()
		c.isListening = true
	}

	event, err := newEvent(c.id, name, args)
	if err != nil {
		panic(err)
	}

	msg, err := event.packBytes()

//	go func() {

		key := ProducerService(name)
		serve:= c.redisClient.conn.SRandMember(key).Val()
//		log.Println("fail: %v",serve)
		if serve!="" {
			serveKey:=ProducerMsgQueen(serve)

			c.redisClient.conn.LPush(serveKey, string(msg[:]))
			task:= ClientTask{
				TaskId:event.MsgId,
				HandlerFunc: handlerFunc,
			}
			atomic.AddInt64(&op, 1)
			log.Println("send: %d",op)
			c.HandleTasks=append(c.HandleTasks,&task)
		}else{
//			log.Println("fail: %v",serve)
		}
//	}()
	return nil
}
var o int64= 0

func (c *Client) Listen() {

	key := ConsumerMsgQueen(c.id)

	for {
		if !c.isListening{
			break;
		}
		msg := c.redisClient.conn.BLPop(2 * time.Second, key).Val()
    	log.Printf("listen:%key: %v\n", key,msg)
		if  len(msg)!=0 {
        	atomic.AddInt64(&o, 1)
			log.Println("e: %d",o)
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

	for i, h := range  c.HandleTasks{
		if h.TaskId == resp.MsgId{
			(*h.HandlerFunc)(resp.RespInfo)
			c.HandleTasks = append(c.HandleTasks[:i], c.HandleTasks[i+1:]...)
			break
		}
	}
	if len(c.HandleTasks)==0 {
		c.isListening = false
	}
}

