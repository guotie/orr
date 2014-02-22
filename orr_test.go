package orr

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
	"strings"
	"testing"
	"time"
)

type FollowUser struct {
	Id int64
	Tm int64
}

type Message struct {
	Id     int64
	Status bool
	ReadTm int64
}

type Account struct {
	Id       int64
	Name     string
	Approved bool
	Reads    map[string]int64
	Messages map[string]Message
	Msg      Message
	Follow   []FollowUser
}

func init() {
	proto := "tcp"
	addr := "127.0.0.1:6379"

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
	_, err := rpool.Get().Do("PING")
	if err != nil {
		panic(err.Error())
	}
	rconn = rpool.Get()
}

func reflectName() {
	acct := Account{}
	pacct := &acct

	fmt.Println(reflect.TypeOf(acct).Name())
	fmt.Println(reflect.TypeOf(acct).Kind())
	fmt.Println(reflect.TypeOf(pacct).Kind())
	fmt.Println(reflect.TypeOf(pacct).Elem().Kind())

	typeName := reflect.TypeOf(pacct).Elem().String()
	fmt.Println(typeName)
	fmt.Println(reverseString(strings.Split(reverseString(typeName), ".")[0]))
}

func reflectValue() {
	var (
		buf []byte = []byte("i am []byte")
		str        = "i am string"
	)

	vb := reflect.ValueOf(buf)
	fmt.Println(vb.Bytes())

	vs := reflect.ValueOf(str)
	fmt.Println(vs.String())
}

func reflectStruct() {
	//acct := Account{}
	//pacct := &acct-+

}

func _TestReflectBasic(t *testing.T) {
	reflectName()
	reflectValue()
}

func saveRedis(key string, id int64, buf []byte) error {
	conn := rpool.Get()
	_, err := conn.Do("HSET", key, id, buf)
	return err
}

func _TestOrrGet(t *testing.T) {
	var i []int = []int{1, 2, 3}
	i = i[1:]
	fmt.Println(i)
	i = i[2:]
	fmt.Println(i)
	fmt.Println(i == nil)
	acct1 := Account{Id: 1}
	acct1.Reads = map[string]int64{"1": 100, "2": 200, "3": 300, "4": 400}
	acct1.Follow = []FollowUser{FollowUser{1, 1}, FollowUser{2, 2}, FollowUser{3, 3}}
	acct1.Messages = map[string]Message{
		"1": Message{1, true, 100},
		"2": Message{2, false, 0},
	}
	acct1.Msg = Message{3, true, 390}

	Save(acct1, "Reads")
	Save(acct1, "Follow")
	Save(acct1, "Messages")

	acct2 := &Account{Id: 1}

	err := Restore(acct2, "Reads")
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println("Reads: ", acct2.Reads)

	err = Restore(acct2, "Messages")
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println("Messages:", acct2.Messages)

	err = Restore(acct2, "Follow")
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println("Follow:", acct2.Follow)

}

func _TestSaveSpeed(t *testing.T) {
	acct2 := &Account{Id: 2}
	acct2.Reads = make(map[string]int64)
	savespeed(acct2, 0)
	savespeed(acct2, 10)
	savespeed(acct2, 100)
	savespeed(acct2, 1000)
	savespeed(acct2, 10000)
}

func savespeed(acct *Account, sz int) {
	for i := 0; i < sz; i++ {
		acct.Reads[string(i)] = int64(i)
	}

	t1 := time.Now()
	Save(acct, "Reads")
	t2 := time.Now()
	d := t2.Sub(t1)
	fmt.Printf("Save %d map: %d nanosecond, %f ms.\n", sz, d, float64(d)/1000000)
}

type F func() int

func tmfunc(f F) (time.Duration, int) {
	t1 := time.Now()
	res := f()
	t2 := time.Now()

	return t2.Sub(t1), res
}

// ------------------------------------------------------
// json reflect marshal tests
// ------------------------------------------------------

type TS struct {
	S []int
	I string
}

func commonMarshal(buf []byte, v interface{}) error {
	err := json.Unmarshal(buf, v)
	return err
}

func reflectMarshal(buf []byte, v interface{}) error {
	err := json.Unmarshal(buf, v)
	return err
}

func testMarshal(t *testing.T) {
	ts1 := TS{S: []int{1, 2, 3, 4, 5, 6, 7}, I: "his"}
	buf, err := json.Marshal(ts1)
	if err != nil {
		t.Fatal("json marshal failed!\n")
	}
	fmt.Println(string(buf))

	ts2 := reflect.New(reflect.TypeOf(ts1)).Interface()
	vts2 := reflect.ValueOf(ts2)
	vpts2 := reflect.ValueOf(&ts2)
	fmt.Println("ts2 Type:", reflect.TypeOf(ts2), ", Kind: ", vts2.Kind(), ", Elem Kind:", vts2.Elem().Kind())
	fmt.Println("&ts2 Type:", reflect.TypeOf(&ts2), ", Kind: ", vpts2.Kind(), ", Elem Kind:", vpts2.Elem().Kind(),
		vpts2.Elem().NumMethod(), ", isNil: ", vpts2.IsNil())
	reflectMarshal(buf, &ts2)
	fmt.Println("ts2 Type:", reflect.TypeOf(ts2))
	fmt.Println(ts2)

	ts3 := ts2.(*TS)
	fmt.Println(ts3)

	ts4 := reflect.New(reflect.TypeOf(ts1)).Elem().Interface()
	vts4 := reflect.ValueOf(ts4)
	vpts4 := reflect.ValueOf(&ts4)
	fmt.Println("ts4 Type:", reflect.TypeOf(ts4), ", Kind: ", vts4.Kind())
	fmt.Println("&ts4 Type:", reflect.TypeOf(&ts4), ", Kind: ", vpts4.Kind(), ", Elem Kind:", vpts4.Elem().Kind(),
		vpts4.Elem().NumMethod(), ", isNil: ", vpts4.IsNil())
	reflectMarshal(buf, &ts4)
	fmt.Println("ts4 Type:", reflect.TypeOf(ts4))
	fmt.Println(ts4)

	si1 := []int{10, 9, 9, 7}
	buf, err = json.Marshal(si1)
	fmt.Println(string(buf))

	si2 := reflect.MakeSlice(reflect.TypeOf(si1), 0, 0).Interface()
	//vsi1 := reflect.TypeOf(&si1)
	fmt.Println("si2 type: ", reflect.TypeOf(si2))
	si2 = si2.([]int)
	fmt.Println("si2 type: ", reflect.TypeOf(si2))
	vsi2 := reflect.ValueOf(&si2)
	fmt.Println("&si2 type: ", reflect.TypeOf(&si2), "Kind: ", vsi2.Kind(),
		"Elem Kind: ", vsi2.Elem().Kind(),
		", Elem Elem Kind: ", vsi2.Elem().Elem().Kind(),
		", NumMethod: ", vsi2.Type().NumMethod())
	reflectMarshal(buf, &si2)
	fmt.Println("si2 type: ", reflect.TypeOf(si2))
	//si2 = reflect.ValueOf(si2).Interface().([]int)
	fmt.Println("si2 decode: ", si2)

	var si3 []int = make([]int, 0)
	vsi3 := reflect.ValueOf(&si3)
	fmt.Println("&si3 type: ", reflect.TypeOf(&si3), "Kind: ", vsi3.Kind(),
		"Elem Kind: ", vsi3.Elem().Kind(),
		", NumMethod: ", vsi3.Type().NumMethod())
	json.Unmarshal(buf, &si3)

	si4 := reflect.New(reflect.TypeOf(si1)).Interface()
	_ = si4
	fmt.Println("si4 type: ", reflect.TypeOf(si4))
	vsi4 := reflect.ValueOf(&si4)
	fmt.Println("&si4 type: ", reflect.TypeOf(&si4), "Kind: ", vsi4.Kind(),
		"Elem Kind: ", vsi4.Elem().Kind(),
		", NumMethod: ", vsi4.Type().NumMethod())
	reflectMarshal(buf, &si4)
	fmt.Println("si4 type: ", reflect.TypeOf(si4))
	//si2 = reflect.ValueOf(si2).Interface().([]int)
	fmt.Println("si4 decode: ", si4)
}
