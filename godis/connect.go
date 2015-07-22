package godis


import
(
     "sync"
     "gopkg.in/redis.v3"
)


type RedisClient struct {
	conn  *redis.Client
	mutex sync.Mutex
}


type Value struct {
	value interface{}
}


func NewRedisClient(host string) *RedisClient {
//	host = fmt.Sprintf("%s:6379", host)
	var conn *redis.Client

	conn = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	client := & RedisClient{
		conn:conn,
		}
	return client
}







