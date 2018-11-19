package healthcheck

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"time"
)

/*
	checks:
	- connection (Pool)
	- closeTest connection
	- ping
*/
func RedisCheck(url string) func() error {
	return func() (err error) {
		pool := &redis.Pool{
			MaxIdle: 1,
			IdleTimeout: 10 * time.Second,
			Dial: func() (redis.Conn, error) { return redis.Dial("tcp", url)},
		}

		conn := pool.Get()
		defer func(error) {
			connErr := conn.Close()

			if err != nil && connErr != nil {
				err = errors.New(fmt.Sprintf("%s\ndb close error: %s", err.Error(), connErr.Error()))
			} else if connErr != nil {
				err = errors.New("db close error: "+connErr.Error())
			}
		}(err)

		data, err := conn.Do("PING")
		if err != nil {
			return errors.New(fmt.Sprintf("%s: %s", "redis ping failed", err.Error()))
		}

		if data == nil {
			return errors.New("empty response for redis ping")
		}

		if data != "PONG" {
			return errors.New(fmt.Sprintf("%s: %s", "unexpected response for redis ping", data))
		}

		data, err = conn.Do("APPEND", "key", "value")
		if err != nil {
			return errors.New(fmt.Sprintf("%s: %s", "redis append failed", data))
		}

		return nil
	}
}
