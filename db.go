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
//     index: 该字段名与结构名(结构名-字段名)，作为辅助hashmap，hashmap的field为该字段值，hashmap的值位obj.Id
//     list: 该字段名与结构名(结构名-字段名)，作为辅助list
//  example: `orr:"index"`

// Insert用于初次数据到redis中，检查struct的tags，并根据tags的定义，
// 来设置redis的辅助字段
// 返回Id
func Insert(obj interface{}) (int64, error) {
	rvobj := reflect.Indirect(reflect.ValueOf(obj))
	rtobj := reflect.TypeOf(rvobj.Interface())

	if rtobj.Kind() != reflect.Struct {
		return -1, fmt.Errorf("Param obj must be struct type.")
	}

	objName := getTypeName(rtobj)
	sid, err := getLatestId("objName" + "Id")
	if err != nil {
		return -1, err
	}
	id, err := strconv.ParseInt(sid, 10, 64)
	if err != nil {
		return -1, err
	}

	buf, err := json.Marshal(obj)
	if err != nil {
		return -1, err
	}

	// 查看结构体是否有辅助字段
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
			if unique(objName+"-"+fieldName, fv) != true {
				return -1, fmt.Errorf("field %s has exist value %s.", fieldName, fieldValue)
			}

			hsetToRedis(objName+"-"+fieldName, fv, []byte(sid), "map")
		} else if tag.Get("orr") == "list" {

		}
	}

	err = hsetToRedis(objName, sid, buf, "map")
	if err != nil {
		return -1, err
	}

	return id, nil
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
