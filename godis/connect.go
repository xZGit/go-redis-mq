package godis


import
(
	"sync"
	"gopkg.in/redis.v3"
)


type RedisClient struct {
	pushConn *redis.Client
	popConn  *redis.Client
	mutex    sync.Mutex
}


type Value struct {
	value interface{}
}


func NewRedisClient(host string) *RedisClient {
	//	host = fmt.Sprintf("%s:6379", host)
	var pushConn, popConn *redis.Client

	pushConn = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0, // use default DB
	})
	popConn = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0, // use default DB
	})
	client := &RedisClient{
		pushConn: pushConn,
		popConn: popConn,
	}
	return client
}







