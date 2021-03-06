package db

import (
	"context"
	"fmt"
	"github.com/gomodule/redigo/redis"
)

type RedisSessionService struct {
	Conn redis.Conn
}

func NewRedisSessionService() *RedisSessionService {
	fmt.Println("Method  NewRedisSessionService (redis)")

	var r = new(RedisSessionService)
	port := "6379"
	var err error
	r.Conn, err = redis.Dial("tcp", "localhost:" + port)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Error in start Redis")
		return nil
	}
	//defer r.Conn.Close()
	fmt.Println("Redis start in port: ", port)
	return r
}

func (r *RedisSessionService) Create(ctx context.Context, userSession *UserSession) (*Nothing, error){
	fmt.Println("Method Create (redis)")


	_, err := r.Conn.Do("SET", userSession.ID, userSession.Email)

	if err != nil {
		fmt.Println(err.Error())
		return &Nothing{}, err
	}

	//defer r.Conn.Close()

	return &Nothing{}, nil
}

func (r *RedisSessionService) Get(ctx context.Context, key *SessionKey) (*SessionValue, error){
	fmt.Println("Method Get (redis)")
	fmt.Println("Data: ", key)

	sessValue := new(SessionValue)
	data, err := r.Conn.Do("GET", key.ID)

	if err != nil {
		fmt.Println(err.Error())
		if  err.Error() == "redigo: nil returned" {
			return nil, nil
		}
		return nil, err
	}

	item, err := redis.String(data, err)
	sessValue.Email = item

	fmt.Println("Data return: ", sessValue)

	return sessValue, nil
}

func (r *RedisSessionService) Delete(ctx context.Context, key *SessionKey) (*Nothing, error){
	fmt.Println("Method Delete (redis)")

	_, err := r.Conn.Do("DEL", key)
	//item, err := redis.String(data, err)

	if err != nil {
		fmt.Println(err.Error())
		return &Nothing{}, err
	}

	//defer r.Conn.Close()

	return &Nothing{}, nil

}
