package multi_position_ring_buffer

import (
	"bytes"
	"encoding/binary"
	"io"
)

func (self *MultiPositionRingBuffer) Write(data []byte) (n int, err error) {
	self.Mu.RLock()
	if self.Err != nil {
		self.Mu.RUnlock()
		return 0, self.Err
	}
	self.Mu.RUnlock()

	// 如果写入数据为空，则返回
	dataLen := len(data)
	if dataLen == 0 {
		// fmt.Println("dataLen == 0")
		return 0, nil
	} else if dataLen <= 8 {
		// 如果写入数据格式不正确，则返回
		// fmt.Println("dataLen <= 8")
		return 0, io.ErrUnexpectedEOF
	}

	// 获取数据序号
	var tmpSeq uint64
	binary.Read(bytes.NewReader(data[:8]), binary.BigEndian, &tmpSeq)
	realData := data[8:]
	realDataLen := len(realData)

	self.Mu.Lock()
	defer self.Mu.Unlock()
	// 根据数据序号判断序号写入位置
	var wBeginPos int
	var avail int
	if self.WSeq == tmpSeq {
		// 如果写入序号从预期写入序号位置开始，则直接0位置写入
		wBeginPos = self.W
		// 判断数据存活空间是否足够
		if avail = self.CheckAvailData(wBeginPos, realDataLen, tmpSeq); avail == 0 {
			// fmt.Println("1 avail == 0", wBeginPos, realDataLen)
			return 0, nil
		}
		// 提前默认写入数据成功，调整所有控制标记位置
		if wBeginPos >= self.R {
			// 如果写入数据在读取数据之前
			c1 := self.Size - self.W
			if c1 >= realDataLen {
				// 如果写入数据长度小于等于剩余缓存长度
				self.W += realDataLen
			} else {
				// 如果写入数据长度大于剩余缓存长度
				self.W = realDataLen - c1
			}
		} else {
			// 如果写入数据在读取数据之后
			self.W += realDataLen
		}
		self.WSeq = tmpSeq + uint64(realDataLen)
		// 判断写入数据后，是否把数据空洞填上了
		if tmpPart, ok := self.WBeginCache[tmpSeq+uint64(realDataLen)]; ok {
			// 如果数据空洞填上了，序号移动到空洞后最后一个位置
			self.W = tmpPart.End
			// fmt.Println("self.W", self.W)
			self.WSeq = tmpPart.EndSeq
			// 清理被填上的数据空洞的序号缓存表
			delete(self.WBeginCache, tmpPart.BeginSeq)
			delete(self.WEndCache, tmpPart.EndSeq)
		}
	} else if self.WSeq < tmpSeq {
		// 如果写入序号比预期写入序号大，则存在数据空洞，跳过数据空洞的部分后写入
		wBeginPos = int((uint64(self.W) + tmpSeq - self.WSeq) % uint64(self.Size))
		// 判断数据存活空间是否足够
		if avail = self.CheckAvailData(wBeginPos, realDataLen, tmpSeq); avail == 0 {
			// fmt.Println("2 avail == 0", wBeginPos, realDataLen)
			return 0, nil
		}
		// 判断序号缓存表是否存在该结束序号。也就是该数据碎片填充之后，是否存在小于该数据碎片的碎片存在。
		if tmpEndPart, ok := self.WEndCache[tmpSeq]; !ok {
			// 如果不存在，判断是否存在开始序号。也就是该数据碎片填充之后，是否存在大于该数据碎片的碎片存在。
			if tmpBeginPart, ok := self.WBeginCache[tmpSeq+uint64(realDataLen)]; !ok {
				// 如果不存在，则在序号缓存表中插入数据空洞碎片
				newPart := &MultiPositionRingPart{
					BeginSeq: tmpSeq,
					EndSeq:   tmpSeq + uint64(realDataLen),
					Begin:    wBeginPos,
					End:      (wBeginPos + realDataLen) % self.Size,
				}
				self.WBeginCache[tmpSeq] = newPart
				self.WEndCache[tmpSeq+uint64(realDataLen)] = newPart
			} else {
				// 如果存在，该碎片与大于该数据碎片的碎片合并
				delete(self.WBeginCache, tmpBeginPart.BeginSeq)
				tmpBeginPart.BeginSeq = tmpSeq
				tmpBeginPart.Begin = wBeginPos
				self.WBeginCache[tmpBeginPart.BeginSeq] = tmpBeginPart
				self.WEndCache[tmpBeginPart.EndSeq] = tmpBeginPart
			}
		} else {
			// 如果存在，判断是否存在开始序号。也就是该数据碎片填充之后，是否存在大于该数据碎片的碎片存在。
			if tmpBeginPart, ok := self.WBeginCache[tmpSeq+uint64(realDataLen)]; !ok {
				// 如果不存在，该碎片与小于该数据碎片的碎片合并
				delete(self.WEndCache, tmpEndPart.EndSeq)
				tmpEndPart.EndSeq = tmpSeq + uint64(realDataLen)
				tmpEndPart.End = (wBeginPos + realDataLen) % self.Size
				self.WEndCache[tmpEndPart.EndSeq] = tmpEndPart
				self.WBeginCache[tmpEndPart.BeginSeq] = tmpEndPart
			} else {
				// 如果存在，让大于该数据碎片的碎片与小于该数据碎片的碎片合并
				delete(self.WEndCache, tmpEndPart.EndSeq)
				delete(self.WBeginCache, tmpBeginPart.BeginSeq)
				tmpEndPart.EndSeq = tmpBeginPart.EndSeq
				tmpEndPart.End = tmpBeginPart.End
				self.WEndCache[tmpEndPart.EndSeq] = tmpEndPart
				self.WBeginCache[tmpEndPart.BeginSeq] = tmpEndPart
			}
		}
	} else {
		// 如果写入序号比预期写入序号小，说明数据已经被写入读取过，则直接返回
		// fmt.Println("self.WSeq < tmpSeq")
		return dataLen, nil
	}

	// 写入数据
	if wBeginPos >= self.R {
		// 如果写入位置在读取位置之后
		c1 := self.Size - wBeginPos
		if c1 >= realDataLen {
			// 如果数据长度小于缓存长度，则直接写入
			copy(self.Buff[wBeginPos:], realData)
		} else {
			// 如果数据长度大于缓存长度，则先写入前半数据，再写入剩余数据
			// fmt.Println("copy:", wBeginPos, c1)
			copy(self.Buff[wBeginPos:], realData[:c1])
			copy(self.Buff[0:], realData[c1:])
		}
	} else {
		// 如果写入位置在读取位置之前
		copy(self.Buff[wBeginPos:], realData)
	}

	// fmt.Println("write ok", dataLen)
	return dataLen, nil
}
