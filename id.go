package orr

import (
	"sync"
)

var idmap = struct {
	sync.Mutex
	ids   map[string]int64
	reuse map[string][]int64
}{
	ids:   make(map[string]int64),
	reuse: make(map[string][]int64),
}

func initID() {

}

func NewId(name string) int64 {
	idmap.Lock()
	defer idmap.Unlock()
	if len(idmap.reuse[name]) > 0 {
		id := idmap.reuse[name][0]
		idmap.reuse[name] = idmap.reuse[name][1:]
		return id
	}

	if id, ok := idmap.ids[name]; ok {
		id++
		idmap.ids[name] = id
		return id
	}

	idmap.ids[name] = 1
	return 1
}

func ReturnId(name string, id int64) {
	idmap.Lock()
	defer idmap.Unlock()

	if idmap.reuse[name] == nil {
		idmap.reuse[name] = make([]int64, 1)
		idmap.reuse[name][0] = id
		return
	}

	idmap.reuse[name] = append(idmap.reuse[name], id)
}

func restoreid() {

}
