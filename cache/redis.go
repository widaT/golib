package cache

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
	"github.com/garyburd/redigo/redis"
)

var (
	DefaultKey = "WebserverRedis"
	Cache = redisCache{}
)

/*
func init()  {
	Cache.StartAndGC(fmt.Sprintf(`{"conn": "%s"}`,config.RedisConn))
}
*/


type redisCache struct {
	p        *redis.Pool // redis connection pool
	conninfo string
	dbNum    int
	key      string
	password string
}


func (rc *redisCache) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	c := rc.p.Get()
	defer c.Close()

	return c.Do(commandName, args...)
}

//SISMEMBER key member
func (rc *redisCache)Sismember(key string,member interface{}) (int,error) {
	return redis.Int(rc.do("SISMEMBER",key,member))
}


func (rc *redisCache)Smembers(key string) ([]string,error) {
	return redis.Strings(rc.do("SMEMBERS",key))
}

func (rc *redisCache)Scard(key string) (int,error) {
	return redis.Int(rc.do("SCARD",key))
}


func (rc *redisCache)Srem(key string,member interface{}) error {
	var err error
	if _, err = rc.do("Srem", key,member); err != nil {
		return err
	}
	return err
}

func (rc *redisCache) Sadd (key string,member interface{})  error {
	var err error
	if _, err = rc.do("SADD", key,  member); err != nil {
		return err
	}
	return nil
}

func (rc *redisCache)Zrange(key string,offset,len int,withscores bool) ([]string,error){
	if withscores {
		return redis.Strings(rc.do("ZRANGE", key, offset, len,"WITHSCORES"))
	}
	return redis.Strings(rc.do("ZRANGE", key, offset, len))
}

func (rc *redisCache)Zrem(key ,member string) (int, error)  {
	return redis.Int(rc.do("ZREM", key, member))
}

func (rc *redisCache) Get(key string) interface{} {
	if v, err := rc.do("GET", key); err == nil {
		return v
	}
	return nil
}

func (rc *redisCache) Lpop(key string) (string,error) {
	return redis.String( rc.do("LPOP", key))
}

func (rc *redisCache) Rpop(key string) (string,error) {
	return redis.String( rc.do("RPOP", key))
}

func (rc *redisCache) GetMulti(keys []string) []interface{} {
	size := len(keys)
	var rv []interface{}
	c := rc.p.Get()
	defer c.Close()
	var err error
	for _, key := range keys {
		err = c.Send("GET", key)
		if err != nil {
			goto ERROR
		}
	}
	if err = c.Flush(); err != nil {
		goto ERROR
	}
	for i := 0; i < size; i++ {
		if v, err := c.Receive(); err == nil {
			rv = append(rv, v.([]byte))
		} else {
			rv = append(rv, err)
		}
	}
	return rv
	ERROR:
	rv = rv[0:0]
	for i := 0; i < size; i++ {
		rv = append(rv, nil)
	}

	return rv
}

func (rc *redisCache) Put(key string, val interface{}, timeout time.Duration) error {
	var err error
	if _, err = rc.do("SETEX", key, int64(timeout / time.Second), val); err != nil {
		return err
	}

	if _, err = rc.do("HSET", rc.key, key, true); err != nil {
		return err
	}
	return err
}

func (rc *redisCache) Set(key string, val interface{}) error {
	var err error
	if _, err = rc.do("SET", key, val); err != nil {
		return err
	}
	return err
}

func (rc *redisCache) Rpush(key string,val interface{} ) error  {
	var err error
	if _, err = rc.do("RPUSH", key, val); err != nil {
		return err
	}
	return err
}

func (rc *redisCache) Lpush(key string,val interface{} ) error  {
	var err error
	if _, err = rc.do("LPUSH", key, val); err != nil {
		return err
	}
	return err
}

func (rc *redisCache) LLen(key string) (int, error)  {
	return redis.Int(rc.do("LLEN",key))
}

func (rc *redisCache) Delete(key string) error {
	var err error
	if _, err = rc.do("DEL", key); err != nil {
		return err
	}
	_, err = rc.do("HDEL", rc.key, key)
	return err
}

func (rc *redisCache) IsExist(key string) bool {
	v, err := redis.Bool(rc.do("EXISTS", key))
	if err != nil {
		return false
	}
	if v == false {
		if _, err = rc.do("HDEL", rc.key, key); err != nil {
			return false
		}
	}
	return v
}

func (rc *redisCache) Incr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, 1))
	return err
}

func (rc *redisCache) Decr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, -1))
	return err
}

func (rc *redisCache) ClearAll() error {
	cachedKeys, err := redis.Strings(rc.do("HKEYS", rc.key))
	if err != nil {
		return err
	}
	for _, str := range cachedKeys {
		if _, err = rc.do("DEL", str); err != nil {
			return err
		}
	}
	_, err = rc.do("DEL", rc.key)
	return err
}


func (rc *redisCache) StartAndGC(config string) error {
	var cf map[string]string
	json.Unmarshal([]byte(config), &cf)

	if _, ok := cf["key"]; !ok {
		cf["key"] = DefaultKey
	}
	if _, ok := cf["conn"]; !ok {
		return errors.New("config has no conn key")
	}
	if _, ok := cf["dbNum"]; !ok {
		cf["dbNum"] = "0"
	}
	if _, ok := cf["password"]; !ok {
		cf["password"] = ""
	}
	rc.key = cf["key"]
	rc.conninfo = cf["conn"]
	rc.dbNum, _ = strconv.Atoi(cf["dbNum"])
	rc.password = cf["password"]

	rc.connectInit()

	c := rc.p.Get()
	defer c.Close()

	return c.Err()
}

func (rc *redisCache) connectInit() {
	dialFunc := func() (c redis.Conn, err error) {
		c, err = redis.Dial("tcp", rc.conninfo)
		if err != nil {
			return nil, err
		}

		if rc.password != "" {
			if _, err := c.Do("AUTH", rc.password); err != nil {
				c.Close()
				return nil, err
			}
		}

		_, selecterr := c.Do("SELECT", rc.dbNum)
		if selecterr != nil {
			c.Close()
			return nil, selecterr
		}
		return
	}
	rc.p = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 180 * time.Second,
		Dial:        dialFunc,
	}
}