package orr

import (
	//	"encoding/json"
	"fmt"
	//"reflect"
	//"strconv"
	"testing"
)

type TmAction struct {
	Tm     int64
	Target int64
	Typ    string
}

type Tuser struct {
	Id        int64
	Name      string `orr:"index"`
	Mobileno  string `orr:"index"`
	Email     string `orr:"index"`
	Password  string `db:"passwd"`
	MainUser  bool   `db:"mainuser"`
	ApproveId string
	IpAddr    string
	DaysLogin int

	Settings map[string]interface{}
	Group    map[string]int64 `json:"-"`
	Thread   []int64          `json:"-"`
	Favor    []TmAction       `json:"-"`
}

func init() {
	OpenRedis("", "")
}

func TestInsert(t *testing.T) {
	u := Tuser{
		Name: "铁哥", Mobileno: "1234", Email: "g@gmail.com"}
	id, err := Insert(&u, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	if id != 1 {
		t.Fatal("id is not 1")
	}
	var u2 Tuser
	err = Select(1, "tuser", &u2)
	if err != nil {
		t.Fatal(err.Error())
	}
	if u2.Id != 1 {
		fmt.Println(u2)
		t.Fatal("Select wrong data")
	}

	fmt.Println(u2)
	Delete(u2)

	reply, err := rconn.Do("INCR", "smth")
	fmt.Println(reply)
}

func TestInsertField(t *testing.T) {
	u := Tuser{
		Name: "铁哥2"}

	u.Group = map[string]int64{"1": 1, "2": 2}
	u.Thread = []int64{1, 2, 3, 4, 5, 6, 7}
	u.Favor = []TmAction{TmAction{10, 10, "thread"}, TmAction{9, 9, "reply"},
		TmAction{8, 8, "user"}, TmAction{7, 7, "group"}}
	if _, err := Insert(&u, true); err != nil {
		t.Fatal(err.Error())
	}

	err := InsertKeyField("hash", "user", "group", u.Id, u.Group)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = InsertKeyField("hash", "user", "thread", u.Id, &u.Thread)
	if err != nil {
		t.Fatal(err.Error())
	}
	/*
		buf, err = json.Marshal(&u.Favor)
		if err != nil {
			t.Fatal(err.Error())
		}
		err = InsertKeyField("hash", "user", "favor", u.Id, buf)
		if err != nil {
			t.Fatal(err.Error())
		}
	*/
	// 从redis中获取数据
	var u2 Tuser
	err = Select(u.Id, "tuser", &u2)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = SelectKeyField("hash", "user", "group", u2.Id, &(u2.Group))
	if err != nil {
		t.Fatal(err.Error())
	}

	err = SelectKeyField("hash", "user", "thread", u2.Id, &(u2.Thread))
	if err != nil {
		t.Fatal(err.Error())
	}

	err = SelectKeyField("hash", "user", "favor", u2.Id, &(u2.Favor))
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(u2)

	Delete(u2)
}
