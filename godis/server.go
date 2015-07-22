package godis


import (
	"sync"
	"log"
	"sync/atomic"
	"time"
)

type ServerTask struct {
	TaskName    string
	HandlerFunc HandleServerFunc
}

type Server struct {
	id          string
	redisClient *RedisClient
	mutex       sync.Mutex
	HandleTasks [] *ServerTask
}

func NewServer(id string, host string) (*Server, error) {
	redisClient := NewRedisClient(host)
	key := Producers()
	val := redisClient.conn.SAdd(key, id).Val()

	if (val==0) {
//		return nil, errors.New("zerorpc/event interface conversion error")
	}

	return &Server{
		id :id,
		redisClient:redisClient,
		HandleTasks:make([]*ServerTask, 0),
	}, nil

}



func (s *Server) RegisterTask(name string, handlerFunc HandleServerFunc, c chan int) {
	go func() {
	log.Println("s %v", s)
		for _, h := range s.HandleTasks {
			if h.TaskName == name {
				log.Println("exist %s", name)
				return
			}
		}
		task := ServerTask{
			TaskName: name,
			HandlerFunc: handlerFunc,
		}
		s.HandleTasks=append(s.HandleTasks, &task)
		key := ProducerService(name)
		s.redisClient.conn.SAdd(key, s.id)
	}()
}

var ops int64= 0
func (s *Server) Listen() {

	key := ProducerMsgQueen(s.id)
	log.Printf(key)
	for {

		msg := s.redisClient.conn.BLPop(2 * time.Second, key).Val()
//		log.Printf("listen:%key: %v\n", key,msg)
		if  len(msg)!=0 {
			atomic.AddInt64(&ops, 1)
			log.Println("rec: %d",ops)
		   go s.ProcessFunc(msg[1])
		}
	}
}


func (s *Server) ProcessFunc(msg string) {

	mySlice := []byte(msg)
	ev, err := unPackEventBytes(mySlice)
	if err != nil {
		panic(err)
	}

	for _, h := range s.HandleTasks {
		if h.TaskName == ev.Name {
			v, err := (*h.HandlerFunc)(ev.Args)
			var resp Resp
			if err != nil {
				resp = newResp(ev.MsgId, 1, err.Error(), nil)
			}else {
				resp = newResp(ev.MsgId, 0, "", v)
			}

			ack, err := resp.packBytes()

			comsumerKey := ConsumerMsgQueen(ev.MId)
			s.redisClient.conn.RPush(comsumerKey, string(ack[:]))
		}
	}

}