package orr

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// 每个结构必须有key键值，字段名必须为Id，必须为正整数
//
// tags:
//   orr
//     index: 该字段名与结构名(结构名_字段名)，作为辅助hashmap，hashmap的field为该字段值，hashmap的值位obj.Id
//     list: 该字段名与结构名(结构名_字段名)，作为辅助list
//  example: `orr:"index"`

// map,list数据过大的解决方法：
//  当map,list数据较少时(1000个以内)，可以将map,list数据保存为json格式; 当map,list数据超过一定规模时，
//  应使用redis的hashmap或list数据结构
//  可以通过设置用户的relations来决定该数据是否以json格式保存

// Insert用于初次数据到redis中，检查struct的tags，并根据tags的定义，
// 来设置redis的辅助字段
// 返回Id
func Insert(obj interface{}) (int64, error) {
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return -1, fmt.Errorf("param obj MUST be type Ptr.")
	}
	rvobj := reflect.ValueOf(obj).Elem()
	rtobj := reflect.TypeOf(rvobj.Interface())

	if rtobj.Kind() != reflect.Struct {
		return -1, fmt.Errorf("Param obj must be struct type.")
	}

	objName := getTypeName(rtobj)

	// 查看结构体是否有辅助字段
	var (
		idxkeys   []string = make([]string, 0)
		idxfields []string = make([]string, 0)
		vid       reflect.Value
	)
	for i := 0; i < rtobj.NumField(); i++ {
		structfield := rtobj.Field(i)
		if structfield.Anonymous {
			continue
		}

		fieldName := structfield.Name
		tag := structfield.Tag
		if tag == "" || tag == "-" {
			continue
		}

		if tag.Get("orr") == "index" {
			fieldValue := rvobj.Field(i)
			if fieldValue.Kind() != reflect.String {
				return -1, fmt.Errorf("index field must be string type!")
			}
			fv := fieldValue.Interface().(string)
			if unique(objName+"_"+fieldName, fv) != true {
				return -1, fmt.Errorf("field %s has exist value %s.", fieldName, fieldValue)
			}
			idxkeys = append(idxkeys, objName+"_"+strings.ToLower(fieldName))
			idxfields = append(idxfields, fv)
		}
	}

	id := NewId(objName)
	sid := strconv.FormatInt(id, 10)

	bsid := []byte(sid)
	for i := 0; i < len(idxkeys); i++ {
		hsetToRedis(idxkeys[i], idxfields[i], bsid, "map")
	}

	vid = rvobj.FieldByName("Id")
	vid.SetInt(id)
	buf, err := json.Marshal(obj)
	if err != nil {
		ReturnId(objName, id)
		return -1, err
	}

	err = hsetToRedis(objName, sid, buf, "map")
	if err != nil {
		for i := 0; i < len(idxkeys); i++ {
			hdelFromRedis(idxkeys[i], idxfields[i], "map")
		}

		ReturnId(objName, id)
		return -1, err
	}

	return id, nil
}

// 将结构体的field插入到数据库中
// typ should be "key" or "hash"
func InsertKeyField(typ, name string, fn string, id int64, value interface{}) error {
	buf, err := json.Marshal(&value)
	if err != nil {
		return err
	}
	sid := strconv.FormatInt(id, 10)
	switch typ {
	case "key":
		_, err := rconn.Do("SET", name+"_"+fn+"_"+sid, buf)
		return err

	case "hash":
		_, err := rconn.Do("HSET", name+"_"+fn, sid, buf)
		return err

	default:
		return fmt.Errorf("param typ invalid, must be key or hash or list.")
	}

	return nil
}

func BuildRelation() {

}

// 仅更新主hashmap
func Update(obj interface{}, objName string, objId int64) error {
	buf, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	err = hsetToRedis(objName, strconv.FormatInt(objId, 10), buf, "map")
	return err
}

func Delete(obj interface{}) error {
	var (
		rvobj reflect.Value
		rtobj reflect.Type
	)

	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		rvobj = reflect.ValueOf(obj).Elem()
		rtobj = reflect.TypeOf(rvobj.Interface())
	} else {
		rvobj = reflect.ValueOf(obj)
		rtobj = reflect.TypeOf(obj)
	}

	if rtobj.Kind() != reflect.Struct {
		return fmt.Errorf("Param obj must be struct type.")
	}

	var (
		idxfields []string
		idxvalue  []string
	)
	objName := getTypeName(rtobj)

	for i := 0; i < rtobj.NumField(); i++ {
		structfield := rtobj.Field(i)
		if structfield.Anonymous {
			continue
		}

		fieldName := structfield.Name
		tag := structfield.Tag
		if tag == "" || tag == "-" {
			continue
		}

		if tag.Get("orr") == "index" {
			idxfields = append(idxfields, objName+"_"+strings.ToLower(fieldName))
			idxvalue = append(idxvalue, rvobj.Field(i).String())
		}
	}

	id := rvobj.FieldByName("Id").Int()
	sid := strconv.FormatInt(id, 10)
	for i, field := range idxfields {
		rconn.Send("HDEL", field, idxvalue[i])
	}
	rconn.Send("HDEL", objName, sid)
	rconn.Flush()
	/*
		for i := 0; i <= len(idxfields); i++ {
			fmt.Println("rconn receive ", i)
			r, err := rconn.Receive()
			if err != nil {
				fmt.Println(err.Error())
			} else {
				fmt.Println(r)
			}
		}
	*/
	return nil
}

func DeleteKeyField(typ string, name string, fn string, id int64) {
	switch typ {
	case "key":
		rconn.Do("DEL", name+"_"+fn+"_"+strconv.FormatInt(id, 10))
		return

	case "hash":
		rconn.Do("HDEL", name+"_"+fn, strconv.FormatInt(id, 10))
		return

	default:
		panic("param typ invalid, must be key or hash or list.")
	}
}

// 从redis中还原数据
func Select(Id int64, name string, res interface{}) error {
	if reflect.TypeOf(res).Kind() != reflect.Ptr {
		return fmt.Errorf("param res must be Ptr type.")
	}

	reply, err := rconn.Do("HGET", name, strconv.FormatInt(Id, 10))
	if err != nil {
		return err
	}

	err = json.Unmarshal(reply.([]byte), res)
	return err
}

func SelectKeyField(keyTyp string, name string,
	fn string, id int64, res interface{}) (err error) {
	if reflect.TypeOf(res).Kind() != reflect.Ptr {
		return fmt.Errorf("param res must be Ptr type.")
	}
	var reply interface{}

	sid := strconv.FormatInt(id, 10)

	switch keyTyp {
	case "key":
		reply, err = rconn.Do("GET", name+"_"+fn+"_"+sid)
	case "hash":
		reply, err = rconn.Do("HGET", name+"_"+fn, sid)
	default:
		return fmt.Errorf("Not support this keytype: %s", keyTyp)
	}
	if err != nil {
		return err
	}
	if reply == nil {
		res = reflect.New(reflect.TypeOf(res)).Elem()
		return
	}

	err = json.Unmarshal(reply.([]byte), res)
	return err
}

func getLatestId(name string) (string, error) {
	reply, err := rconn.Do("INCR", name)
	if err != nil {
		return "", err
	}

	return reply.(string), nil
}

func parseTag(str string) (typ string, addational_typ string) {
	if str == "-" {
		typ = str
	} else if str != "" {
		tags := strings.Split(str, ";")
		m := make(map[string]string)
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
			if len(v) == 2 {
				m[k] = v[1]
			} else {
				m[k] = k
			}
		}

		addational_typ = m["NOT NULL"] + " " + m["UNIQUE"]
	}
	return
}

func unique(name string, value string) bool {
	reply, _ := rconn.Do("HGET", name, value)
	if reply != nil {
		return false
	}

	return true
}
