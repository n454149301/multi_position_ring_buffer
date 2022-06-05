package multi_position_ring_buffer

import "sync/atomic"

// 判断数据存活空间是否足够
func (self *MultiPositionRingBuffer) CheckAvailData(wBeginPos int32, realDataLen int32, tmpSeq uint64) (avail int32) {
	if atomic.LoadInt32(&self.W) > atomic.LoadInt32(&self.R) {
		avail = atomic.LoadInt32(&self.Size) - int32(wBeginPos) + atomic.LoadInt32(&self.R)
	} else if atomic.LoadInt32(&self.W) < atomic.LoadInt32(&self.R) {
		avail = atomic.LoadInt32(&self.R) - wBeginPos
	} else {
		if atomic.LoadUint64(&self.RSeq) == atomic.LoadUint64(&self.WSeq) {
			// 如果读序号和写序号重合，说明写入数据全部读取了，那么缓存区是空的
			avail = atomic.LoadInt32(&self.Size)
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
		if (tmpSeq + uint64(realDataLen) - atomic.LoadUint64(&self.RSeq)) > uint64(atomic.LoadInt32(&self.Size)) {
			avail = 0
			return
		}
		return
	}
}
