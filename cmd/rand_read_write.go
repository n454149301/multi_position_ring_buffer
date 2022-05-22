package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/n454149301/multi_position_ring_buffer"
)

func main() {
	tmpBuf := multi_position_ring_buffer.New(100)
	end := make(chan bool, 1)

	// 读取测试
	go func() {
		rm := sync.RWMutex{}
		rNum := 0
		for {
			// time.Sleep(time.Second / 100)
			readTmp := make([]byte, 100)
			if readNum, err := tmpBuf.Read(readTmp); err != nil {
				panic(err)
			} else if readNum == 0 {
				// time.Sleep(time.Second / 100)
				continue
			} else {
				fmt.Printf("readNum: %d %s", readNum, string(readTmp))
				if readNum%7 != 0 {
					panic(fmt.Sprintf("readNum: %d %s", readNum, string(readTmp)))
				}
				rm.Lock()
				rNum += readNum
				/*
					if rNum > 980000 {
						fmt.Println("read:", string(readTmp), rNum, readNum, readNum/7)
						// tmpBuf.Close()
						fmt.Println("close buf", tmpBuf.Err)
						rm.Unlock()
						end <- true
						break
					}
				*/
				rm.Unlock()
				fmt.Println("read:", string(readTmp), rNum, readNum, readNum/7)
			}
		}
	}()

	// 写入测试
	muMap := map[uint64]bool{}
	mu := sync.RWMutex{}
	bigNum := uint64(0)

	go func() {
		for {
			tmpRandNum := uint64(rand.Intn(14))
			mu.Lock()
			muMap[tmpRandNum] = true
			mu.Unlock()

			// test data
			dataSeqByte := make([]byte, 8)
			binary.BigEndian.PutUint64(dataSeqByte, (tmpRandNum*7 + bigNum*14))
			tmpData := bytes.Join([][]byte{dataSeqByte, []byte("1234567")}, []byte{})

			if wNum, err := tmpBuf.Write(tmpData); err != nil {
				panic(err)
			} else if wNum == 0 {
				// 缓存区满了
				time.Sleep(time.Second / 1000)
				continue
			}

			mu.Lock()
			if len(muMap) == 14 {
				muMap = map[uint64]bool{}
				bigNum++
				fmt.Println("end", bigNum)
				if bigNum >= 10000 {
					mu.Unlock()
					break
				}
			}
			mu.Unlock()
			// fmt.Println(tmpBuf.Buff, tmpBuf.Size)
			// time.Sleep(time.Second / 100)
		}
	}()

	<-end
}
