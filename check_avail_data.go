package multi_position_ring_buffer

// 判断数据存活空间是否足够
func (self *MultiPositionRingBuffer) CheckAvailData(wBeginPos int, realDataLen int, tmpSeq uint64) (avail int) {
	if self.W > self.R {
		avail = self.Size - wBeginPos + self.R
	} else if self.W < self.R {
		avail = self.R - wBeginPos
	} else {
		if self.RSeq == self.WSeq {
			// 如果读序号和写序号重合，说明写入数据全部读取了，那么缓存区是空的
			avail = self.Size
		} else {
			avail = 0
		}
	}

	if realDataLen > avail {
		// 如果数据存活空间不足，则返回0
		avail = 0
		return
	} else {
		// 如果写入序号加数据长度已经超过剩余一轮缓存区大小，说明缓存区已经不能继续写入了。
		if (tmpSeq + uint64(realDataLen) - self.RSeq) > uint64(self.Size) {
			avail = 0
			return
		}
		return
	}
}
