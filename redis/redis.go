// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package redis for cache provider
//
// depend on github.com/gomodule/redigo/redis
//
// go install github.com/gomodule/redigo/redis
//
// Usage:
// import(
//   _ "github.com/astaxie/beego/cache/redis"
//   "github.com/astaxie/beego/cache"
// )
//
//  bm, err := cache.NewCache("redis", `{"conn":"127.0.0.1:11211"}`)
//
//  more docs http://beego.me/docs/module/cache.md
package cache

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
	"github.com/garyburd/redigo/redis"
)

type Redis struct {
	p        *redis.Pool
	conninfo string
	dbNum    int
	key      string
	password string
}


func (rc *Redis) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	c := rc.p.Get()
	defer c.Close()

	return c.Do(commandName, args...)
}

//SISMEMBER key member
func (rc *Redis)Sismember(key string,member interface{}) (int,error) {
	return redis.Int(rc.do("SISMEMBER",key,member))
}


func (rc *Redis)Smembers(key string) ([]string,error) {
	return redis.Strings(rc.do("SMEMBERS",key))
}

func (rc *Redis)Scard(key string) (int,error) {
	return redis.Int(rc.do("SCARD",key))
}


func (rc *Redis)Srem(key string,member interface{}) error {
	var err error
	if _, err = rc.do("Srem", key,member); err != nil {
		return err
	}
	return err
}

func (rc *Redis) Sadd (key string,member interface{})  error {
	var err error
	if _, err = rc.do("SADD", key,  member); err != nil {
		return err
	}
	return nil
}

func (rc *Redis)Zrange(key string,start,stop int,withscores bool) ([]string,error){
	if withscores {
		return redis.Strings(rc.do("ZRANGE", key, start, stop,"WITHSCORES"))
	}
	return redis.Strings(rc.do("ZRANGE", key, start, stop))
}

func (rc *Redis)Zrem(key ,member string) (int, error)  {
	return redis.Int(rc.do("ZREM", key, member))
}

func (rc *Redis) Get(key string) interface{} {
	if v, err := rc.do("GET", key); err == nil {
		return v
	}
	return nil
}

func (rc *Redis) Lpop(key string) (string,error) {
	return redis.String( rc.do("LPOP", key))
}

func (rc *Redis) Rpop(key string) (string,error) {
	return redis.String( rc.do("RPOP", key))
}

func (rc *Redis) GetMulti(keys []string) []interface{} {
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

func (rc *Redis) Put(key string, val interface{}, timeout time.Duration) error {
	var err error
	if _, err = rc.do("SETEX", key, int64(timeout / time.Second), val); err != nil {
		return err
	}

	if _, err = rc.do("HSET", rc.key, key, true); err != nil {
		return err
	}
	return err
}

func (rc *Redis) Set(key string, val interface{}) error {
	var err error
	if _, err = rc.do("SET", key, val); err != nil {
		return err
	}
	return err
}

func (rc *Redis) Rpush(key string,val interface{} ) error  {
	var err error
	if _, err = rc.do("RPUSH", key, val); err != nil {
		return err
	}
	return err
}

func (rc *Redis) Lpush(key string,val interface{} ) error  {
	var err error
	if _, err = rc.do("LPUSH", key, val); err != nil {
		return err
	}
	return err
}

func (rc *Redis) LLen(key string) (int, error)  {
	return redis.Int(rc.do("LLEN",key))
}

func (rc *Redis) Delete(key string) error {
	var err error
	if _, err = rc.do("DEL", key); err != nil {
		return err
	}
	_, err = rc.do("HDEL", rc.key, key)
	return err
}

func (rc *Redis) IsExist(key string) bool {
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

func (rc *Redis) Incr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, 1))
	return err
}

func (rc *Redis) Decr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, -1))
	return err
}

func (rc *Redis)Hset(key ,member string,val interface{}) bool {
	if _, err := rc.do("HSET", key, member,val); err != nil {
		return false
	}
	return true
}

func (rc *Redis)Hdel(key ,member string) bool {
	if _, err := rc.do("HDEL", key, member); err != nil {
		return false
	}
	return true
}

func (rc *Redis) NewRedis(config string) (*Redis,error) {
	var cf map[string]string
	json.Unmarshal([]byte(config), &cf)
	if _, ok := cf["conn"]; !ok {
		return nil,errors.New("config has no conn key")
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
	return rc,c.Err()
}

func (rc *Redis) connectInit() {
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