package godis


import (
	"sync"
	"log"
	"time"
	"gopkg.in/redis.v3"
)



type Server struct {
	id          string
	redisClient *RedisClient
	mutex       sync.Mutex
	HandleTasks map[string]HandleServerFunc
}

func NewServer(id string, addr string) (*Server, error) {
	redisClient, err := NewRedisClient(addr)
	if err!=nil {
		return nil, err
	}
	key := Producers(id)
	val := redisClient.pushConn.Keys(key)
	if val.Err()!=nil {
		return nil, val.Err()
	}
	if len(val.Val())!=0 {
		return nil, SCRepeatErr
	}
	key =  ProducerService(id)
	redisClient.pushConn.Del(key)
	redisClient.pushConn.Set(key, 1, 120*time.Second)
	return &Server{
		id :id,
		redisClient:redisClient,
		HandleTasks:make(map[string]HandleServerFunc),
	}, nil

}



func (s *Server) RegisterTask(name string, handlerFunc HandleServerFunc) {

	go func() {
		if _, ok := s.HandleTasks[name]; ok {
			log.Println("register repeat service %s", name)
			panic(ServiceRepeatErr)
			return
		}
		s.HandleTasks[name] = handlerFunc
	}()

}






func (s *Server) Listen() {

	key := ProducerMsgQueen(s.id)
	go s.HeartBeat()
	for {

		msg := s.redisClient.popConn.BLPop(2 * time.Second, key)
		if len(msg.Val())!=0 {
			go s.ProcessFunc(msg.Val()[1])
			go s.IncServerDealCount()
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
			m := redis.Z{
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
		panic(err.Error())
		return
	}
	var resp Resp

	if f, ok := s.HandleTasks[ev.Name]; ok {
		code, data, err := (*f)(ev.Args)
		msg := ""
		if (err!=nil) {
			msg =err.Error()
		}
		resp = newResp(ev.MsgId, code, msg, data)
	}else {
		resp = newResp(ev.MsgId, 1002, errMsgMap[1002], nil)
	}

	ack, err := resp.packBytes()
	if err!=nil {
		panic(err.Error())
		return
	}
	consumerKey := ConsumerMsgQueen(ev.MId)
	s.redisClient.pushConn.RPush(consumerKey, string(ack[:]))


}

func (s *Server) IncServerDealCount() {
	key := ServerDealCount(s.id)
	s.redisClient.pushConn.Incr(key)
}


