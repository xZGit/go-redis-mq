package godis


import (
	"sync"
	"log"
	"sync/atomic"
	"time"
	"gopkg.in/redis.v3"
)



type Server struct {
	id          string
	redisClient *RedisClient
	mutex       sync.Mutex
	HandleTasks map[string]HandleServerFunc
}

func NewServer(id string, host string) (*Server, error) {
	redisClient := NewRedisClient(host)
	key := Producers(id)
	val := redisClient.pushConn.Keys(key).Val()
	redisClient.pushConn.Set(key, 1, 60*time.Second)
	log.Println("key,%v", val)
	return &Server{
		id :id,
		redisClient:redisClient,
		HandleTasks:make(map[string]HandleServerFunc),
	}, nil

}



func (s *Server) RegisterTask(name string, handlerFunc HandleServerFunc, c chan int) {
	go func() {
		log.Println("s %v", s)
		if _, ok := s.HandleTasks[name]; ok {
			log.Println("register repeat %s", name)
			return
		}
		s.HandleTasks[name] = handlerFunc

	}()
}





var ops int64 = 0
func (s *Server) Listen() {

	key := ProducerMsgQueen(s.id)
	log.Printf(key)
	go s.HeartBeat()
	for {

		msg := s.redisClient.popConn.BLPop(2 * time.Second, key).Val()
		//		log.Printf("listen:%key: %v\n", key,msg)
		if len(msg)!=0 {
			atomic.AddInt64(&ops, 1)
			log.Println("rec: %d", ops)
			go s.ProcessFunc(msg[1])
		}
	}
}

func (s *Server) HeartBeat() {

	serverKey := Producers(s.id)

	var beatFunc = func() {

		expireTime := time.Now().Add(120*time.Second)
		s.redisClient.popConn.ExpireAt(serverKey, expireTime)
		for key, _ := range s.HandleTasks {
			serviceKey := ProducerService(key)
			log.Println("register key:", serviceKey)
			m:= redis.Z{
				Score:float64(expireTime.Unix()),
				Member:s.id,
			}
			s.redisClient.popConn.ZAdd(serviceKey, m)
		}

	}

	beatFunc()

	timer := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-timer.C:
			go beatFunc()
		}
	}
}


func (s *Server) ProcessFunc(msg string) {

	mySlice := []byte(msg)
	ev, err := unPackEventBytes(mySlice)
	if err != nil {
		panic(err)
	}
	var resp Resp

	if f, ok := s.HandleTasks[ev.Name]; ok {
		v, err := (*f)(ev.Args)

		if err != nil {
			resp = newResp(ev.MsgId, 1, "err", nil)
		}else {
			resp = newResp(ev.MsgId, 0, "", v)
		}

	}else {
		resp = newResp(ev.MsgId, 0, "", nil)
	}
	ack, err := resp.packBytes()
	consumerKey := ConsumerMsgQueen(ev.MId)
	s.redisClient.pushConn.RPush(consumerKey, string(ack[:]))


}