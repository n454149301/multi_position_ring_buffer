package multi_position_ring_buffer

/*
#include <stdlib.h>
*/
import "C"

import (
	"io"
	"sync"
	"sync/atomic"
	"unsafe"
)

type MultiPositionRingPart struct {
	BeginSeq uint64
	EndSeq   uint64
	Begin    int32
	End      int32
}

type MultiPositionRingBuffer struct {
	BuffP unsafe.Pointer
	Buff  []byte
	Size  int32
	R     int32
	RSeq  uint64
	W     int32
	// key为序号断点开始位置，value为循环缓存区一个断点结构
	WBeginCache map[uint64]*MultiPositionRingPart
	// key为序号断点结束位置，value为循环缓存区一个断点结构
	WEndCache map[uint64]*MultiPositionRingPart
	WSeq      uint64
	Mu        sync.RWMutex

	// 添加状态锁，如果没有人写入，则读被阻塞
	ChLock chan bool

	// Err error
	Err atomic.Value
}

func New(size int) *MultiPositionRingBuffer {
	p := C.malloc(C.ulong(size))
	return &MultiPositionRingBuffer{
		BuffP:       p,
		Buff:        C.GoBytes(p, C.int(size)),
		Size:        int32(size),
		R:           0,
		W:           0,
		WBeginCache: make(map[uint64]*MultiPositionRingPart),
		WEndCache:   make(map[uint64]*MultiPositionRingPart),
		WSeq:        0,
		ChLock:      make(chan bool, 1),
	}
}

func (self *MultiPositionRingBuffer) Close() {
	self.Mu.Lock()
	defer self.Mu.Unlock()
	if self.BuffP != nil {
		C.free(self.BuffP)
		self.BuffP = nil
	}

	if len(self.ChLock) == 0 {
		// fmt.Println("self.ChLock lock begin")
		self.ChLock <- true
		// fmt.Println("self.ChLock lock end")
	}

	self.Err.Store(&io.ErrClosedPipe)
}

func (self *MultiPositionRingBuffer) Flush() {
	if len(self.ChLock) == 0 {
		// fmt.Println("self.ChLock lock begin")
		self.ChLock <- true
		// fmt.Println("self.ChLock lock end")
	}
}
