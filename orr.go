package orr

// object reflect to redis
// currently, mainly support map, slice, array, struct

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
	//"strconv"
	"strings"
	"time"
)

var (
	rpool *redis.Pool
	rconn redis.Conn
)

func OpenRedis(proto, addr string) {
	if proto == "" {
		proto = "tcp"
	}
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rpool = &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 600 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(proto, addr)
			if err != nil {
				panic(err)
				return nil, err
			}

			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	rconn = rpool.Get()
	_, err := rconn.Do("PING")
	if err != nil {
		panic(err.Error())
	}
}

// 将数据结构Marshal, 并保存
func Save(obj interface{}, fieldname string) error {
	if fieldname[0] < 'A' || fieldname[0] > 'Z' {
		return fmt.Errorf("Param obj's field %s should be exported.\n", fieldname)
	}

	tobj := reflect.TypeOf(obj)
	vobj := reflect.ValueOf(obj)

	if tobj.Kind() == reflect.Ptr {
		vobj = vobj.Elem()
		tobj = reflect.TypeOf(vobj)
	}

	if tobj.Kind() != reflect.Struct {
		return errors.New("Param obj must be Struct or Ptr of Struct.")
	}

	vfield := vobj.FieldByName(fieldname)
	if !vfield.IsValid() {
		return errors.New("Param obj's field " + fieldname + " is invalid.")
	}

	vid := vobj.FieldByName("Id")
	if !vid.IsValid() {
		return errors.New("Param obj must has Id field.")
	}
	iid := vid.Int()

	typName := getTypeName(tobj)
	if strings.Contains(typName, "-") {
		return errors.New("Struct name should not contains -.")
	}

	redisFieldname := typName + "_" + strings.ToLower(fieldname)

	buf, err := json.Marshal(vfield.Interface())
	if err != nil {
		return fmt.Errorf("Marshal field %s failed: %s.", err.Error())
	}
	err = hsetToRedis(redisFieldname, iid, buf, "map")
	if err != nil {
		return err
	}

	return nil
}

// 从redis中恢复数据
func Restore(obj interface{}, fieldname string) error {
	if fieldname[0] < 'A' || fieldname[0] > 'Z' {
		return fmt.Errorf("Param obj's field %s should be exported.\n", fieldname)
	}

	tobj := reflect.TypeOf(obj)

	if tobj.Kind() != reflect.Ptr {
		return errors.New("Param obj must be Ptr.\n")
	}
	if tobj.Elem().Kind() != reflect.Struct {
		return errors.New("Param obj must be Ptr of Struct.\n")
	}

	vobj := reflect.ValueOf(obj).Elem()
	vfield := vobj.FieldByName(fieldname)
	if !vfield.IsValid() {
		return errors.New("Param obj's field " + fieldname + " is invalid.\n")
	}

	vid := vobj.FieldByName("Id")
	if !vid.IsValid() {
		return errors.New("Param obj must has Id field.\n")
	}
	iid := vid.Int()

	typName := getTypeName(tobj)
	redisFieldname := typName + "_" + strings.ToLower(fieldname)

	if !vfield.CanSet() {
		return fmt.Errorf("Param obj field %s cannot be set.\n", fieldname)
	}

	res, err := hgetFromRedis(redisFieldname, iid, vfield.Type())
	if err != nil {
		return fmt.Errorf("Get obj's field %s data from redis failed: %s.\n",
			fieldname, err.Error())
	}
	vfield.Set(reflect.ValueOf(res).Elem())

	return nil
}

// 向slice或map中追加一个元素, 并保存
// 当map时, 使用HSET
// 当slice时, 使用RPUSH
func AppElement(obj interface{}, fieldname string, args ...interface{}) {
	/*
		if fieldname[0] < 'A' || fieldname[0] > 'Z' {
			return fmt.Errorf("Param obj's field %s should be exported.\n", fieldname)
		}

		tobj := reflect.TypeOf(obj)
		vobj := reflect.ValueOf(obj)

		if tobj.Kind() == reflect.Ptr {
			vobj = vobj.Elem()
			tobj = reflect.TypeOf(vobj)
		}

		if tobj.Kind() != reflect.Struct {
			return errors.New("Param obj must be Struct or Ptr of Struct.\n")
		}
	*/
}

// 从slice或map中删除一个元素, 并保存
func RemElement(obj interface{}, fieldname string, args ...interface{}) error {
	if fieldname[0] < 'A' || fieldname[0] > 'Z' {
		return fmt.Errorf("Param obj's field %s should be exported.\n", fieldname)
	}
	return nil
}

func hgetFromRedis(key string, field interface{}, typ reflect.Type) (interface{}, error) {
	res := reflect.New(typ).Interface()

	conn := rpool.Get()
	defer conn.Close()
	buf, err := conn.Do("HGET", key, field)
	if err != nil {
		return nil, err
	}
	if buf == nil {
		return nil, fmt.Errorf("value of key %s & field %q is nil.\n", key, field)
	}
	err = json.Unmarshal(buf.([]byte), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func hsetToRedis(key string, field interface{}, value []byte, typ string) error {
	//conn := rpool.Get()
	//defer conn.Close()
	_, err := rconn.Do("HSET", key, field, value)
	if err != nil {
		return err
	}

	return nil
}

func hdelFromRedis(key string, field interface{}, typ string) {
	rconn.Do("HDEL", key, field)
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// 返回Type的name
func getTypeName(typ reflect.Type) string {
	var typeName string
	if typ.Kind() == reflect.Ptr {
		typeName = typ.Elem().String()
	} else {
		typeName = typ.String()
	}

	names := strings.Split(typeName, ".")
	if len(names) == 1 {
		return strings.ToLower(names[0])
	}

	return strings.ToLower(names[len(names)-1])
}
