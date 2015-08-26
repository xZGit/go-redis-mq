package godis


import (
	"sync"
	"gopkg.in/redis.v3"
	"time"
	"log"
)



type Monitor struct {
	conn  *redis.Client
	mutex sync.Mutex

}

func NewMonitor(addr string) (*Monitor, error) {
	var conn *redis.Client

	conn = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0, // use default DB
	})

	monitor := &Monitor{
		conn:conn,
	}
	return monitor, nil

}


func (m *Monitor) Listen() {
	log.Println("start monitor")
	timer := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-timer.C:
			go m.Log()
		}
	}
}



func (m *Monitor) Log() {

	log.Println("time now is:", time.Now().Format("2006-01-02 15:04:05"))

	var logSliceKey = func(key string) {

		keys := m.conn.Keys(key).Val()
		for _, v := range keys {
			log.Println(v, m.conn.Get(v).Val())
		}
	}
	key := ServerDealCount("*")
	logSliceKey(key)
	key = ServiceCallCount("*")
	logSliceKey(key)
	key = ServiceNotFoundCount("*")
	logSliceKey(key)
	key = RequestTimeoutCount("*")
	logSliceKey(key)
}