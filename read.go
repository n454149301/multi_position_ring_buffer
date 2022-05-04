package multi_position_ring_buffer

import (
	"io"
)

func (self *MultiPositionRingBuffer) Read(data []byte) (n int, err error) {
	self.Mu.RLock()
	if self.Err != nil {
		err = self.Err
		self.Mu.RUnlock()
		return
	}

	// 如果读取buff小于缓存区大小，返回错误
	if len(data) < self.Size {
		self.Mu.RUnlock()
		return 0, io.ErrShortBuffer
	}

	// 如果读写序号相同，则说明缓冲区为空
	if self.RSeq == self.WSeq {
		self.Mu.RUnlock()
		return 0, nil
	}

	self.Mu.RUnlock()
	self.Mu.Lock()
	defer self.Mu.Unlock()
	if self.W > self.R {
		// 如果写位置在读前面，直接读取
		n = self.W - self.R
		// fmt.Println("read w, r:", self.W, self.R)
		copy(data[0:], self.Buff[self.R:self.R+n])
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
		n = c1 + self.W

		self.R = self.W
		self.RSeq += uint64(n)
		return n, nil
	}
}
