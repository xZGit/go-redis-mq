package godis


import(
	"sync"
    "log"
	"time"
//	"errors"
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

	go func() {

		key := ProducerService(name)
		serve:= c.redisClient.conn.SRandMember(key).Val()

		if serve!="" {
			serveKey:=ProducerMsgQueen(serve)
//			log.Printf("serverKey: %v\n", serveKey)
			c.redisClient.conn.LPush(serveKey, string(msg[:]))
			task:= ClientTask{
				TaskId:event.MsgId,
				HandlerFunc: handlerFunc,
			}

			c.HandleTasks=append(c.HandleTasks,&task)
		}
	}()
	return nil
}


func (c *Client) Listen() {

	key := ConsumerMsgQueen(c.id)

	for {
		msg := c.redisClient.conn.BLPop(2 * time.Second, key).Val()
		log.Printf("listen:%key: %v\n", key,msg)
		if  len(msg)!=0 {
//			log.Printf("msg: %v\n", msg)
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

	for _, h := range  c.HandleTasks{
		if h.TaskId == resp.MsgId{
			(*h.HandlerFunc)(resp.RespInfo)
		}
	}

}

