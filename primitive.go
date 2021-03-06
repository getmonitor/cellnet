package cellnet

import (
	"log"
	"sync"
)

var (
	// Cell实例管理
	cellMapGuard sync.RWMutex
	cellMap      map[CellID]*cell = make(map[CellID]*cell)

	// CellID生成器
	indexAccGuard sync.RWMutex
	indexAcc      int32
)

func genID() CellID {

	indexAccGuard.Lock()
	defer indexAccGuard.Unlock()

	// TODO 处理翻越case
	indexAcc++

	return NewCellID(RegionID, indexAcc)
}

func findCell(id CellID) *cell {
	cellMapGuard.RLock()
	defer cellMapGuard.RUnlock()

	if v, ok := cellMap[id]; ok {
		return v
	}

	return nil
}

// CellID是否为本进程内的ID
func IsLocal(id CellID) bool {
	return id.Region() == RegionID
}

// 为消息处理函数生成一个Cell, 返回CellID
func Spawn(callback func(CellID, interface{})) CellID {

	id := genID()

	if config.CellLog {
		log.Println("[cellnet] #spawn", id.String(), GetStackInfoString(2))
	}

	c := &cell{
		mailbox: make(chan interface{}, 8),
		id:      id,
	}

	cellMapGuard.Lock()
	cellMap[id] = c
	cellMapGuard.Unlock()

	go func() {

		for {

			if data, ok := c.fetch(); ok {
				callback(id, data)
			} else {
				break
			}

		}

	}()

	c.post(EventInit{})

	return id
}

// 将制定内容发送到target的Cell中
func Send(target CellID, data interface{}) bool {

	if target == 0 {
		return false
	}

	if IsLocal(target) {
		return SendLocal(target, data)
	}

	if expressDriver == nil {

		log.Println("[cellnet] express func nil, target not send", target.String())

		return false
	}

	if !expressDriver(target, data) {

		log.Println("[cellnet] extern target not found: ", target.String())
		return false
	}

	return true
}

// 将制定内容发送到本地的target的Cell中
func SendLocal(target CellID, data interface{}) bool {
	if c := findCell(target); c != nil {

		if config.CellLog {
			log.Printf("[cellnet] #send %v %v %v", target.String(), ReflectContent(data), GetStackInfoString(2))
		}

		c.post(data)
		return true
	}

	log.Println("[cellnet] target not found: ", target.String())

	return false
}

var expressDriver func(CellID, interface{}) bool

// 设置快递驱动, 负责将给定内容跨进程送达
func SetExpressDriver(driver func(CellID, interface{}) bool) {
	expressDriver = driver
}
