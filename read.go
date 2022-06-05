package multi_position_ring_buffer

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
)

func (self *MultiPositionRingBuffer) Read(data []byte) (n int, err error) {
	errI := self.Err.Load()
	if errI != nil {
		var coverErrOk bool
		if err, coverErrOk = errI.(error); coverErrOk {
			return
		} else {
			err = errors.New("Load error cover failed:" + fmt.Sprint(err))
			return
		}
	}

	// 如果读取buff小于缓存区大小，返回错误
	if int32(len(data)) < atomic.LoadInt32(&self.Size) {
		return 0, io.ErrShortBuffer
	}

	// 如果读写序号相同，则说明缓冲区为空
	if atomic.LoadUint64(&self.RSeq) == atomic.LoadUint64(&self.WSeq) {
		return 0, nil
	}

	self.Mu.Lock()
	defer self.Mu.Unlock()
	if self.W > self.R {
		// 如果写位置在读前面，直接读取
		n = int(self.W) - int(self.R)
		// fmt.Println("read w, r:", self.W, self.R)
		copy(data[0:], self.Buff[int(self.R):int(self.R)+n])
		self.R = self.W
		self.RSeq += uint64(n)
		return
	} else {
		// 如果写位置在读后面或者相同，到buff结尾读取为数据前半段，从0位置到W位置是数据后半段
		c1 := self.Size - self.R
		copy(data[0:], self.Buff[self.R:])
		if self.W != 0 {
			copy(data[c1:], self.Buff[:self.W])
		}
		n = int(c1) + int(self.W)

		self.R = self.W
		self.RSeq += uint64(n)
		return n, nil
	}
}
