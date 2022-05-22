package multi_position_ring_buffer

/*
#include <stdlib.h>
*/
import "C"

import (
	"io"
	"sync"
	"unsafe"
)

type MultiPositionRingPart struct {
	BeginSeq uint64
	EndSeq   uint64
	Begin    int
	End      int
}

type MultiPositionRingBuffer struct {
	BuffP unsafe.Pointer
	Buff  []byte
	Size  int
	R     int
	RSeq  uint64
	W     int
	// key为序号断点开始位置，value为循环缓存区一个断点结构
	WBeginCache map[uint64]*MultiPositionRingPart
	// key为序号断点结束位置，value为循环缓存区一个断点结构
	WEndCache map[uint64]*MultiPositionRingPart
	WSeq      uint64
	Mu        sync.RWMutex

	Err error
}

func New(size int) *MultiPositionRingBuffer {
	p := C.malloc(C.ulong(size))
	return &MultiPositionRingBuffer{
		BuffP:       p,
		Buff:        C.GoBytes(p, C.int(size)),
		Size:        size,
		R:           0,
		W:           0,
		WBeginCache: make(map[uint64]*MultiPositionRingPart),
		WEndCache:   make(map[uint64]*MultiPositionRingPart),
		WSeq:        0,
	}
}

func (self *MultiPositionRingBuffer) Close() {
	self.Mu.Lock()
	C.free(self.BuffP)
	self.Err = io.ErrClosedPipe
	self.Mu.Unlock()
}
