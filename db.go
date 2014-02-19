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
	rvobj := reflect.Indirect(reflect.ValueOf(obj))
	rtobj := reflect.TypeOf(rvobj.Interface())

	if rtobj.Kind() != reflect.Struct {
		return -1, fmt.Errorf("Param obj must be struct type.")
	}

	buf, err := json.Marshal(obj)
	if err != nil {
		return -1, err
	}

	objName := getTypeName(rtobj)

	// 查看结构体是否有辅助字段
	var (
		idxkeys   []string = make([]string, 0)
		idxfields []string = make([]string, 0)
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
			idxkeys = append(idxkeys, objName+"_"+fieldName)
			idxfields = append(idxfields, fv)
		}
	}

	id := NewId(objName)
	sid := strconv.FormatInt(id, 10)

	bsid := []byte(sid)
	for i := 0; i < len(idxkeys); i++ {
		hsetToRedis(idxkeys[i], idxfields[i], bsid, "map")
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
	return nil
}

// 从redis中还原数据
func Select(Id int64, name string, args ...interface{}) interface{} {
	return nil
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
