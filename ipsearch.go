package ipsearch

import (
	"errors"
	"io/ioutil"
	"strconv"
	"strings"
)

type ipIndex struct {
	startIP, endIP           uint32
	localOffset, localLength uint32
}

type prefixIndex struct {
	startIndex, endIndex uint32
}

type IPSearch struct {
	data               []byte
	prefixMap          map[uint32]prefixIndex
	firstStartIpOffset uint32
	prefixStartOffset  uint32
	prefixEndOffset    uint32
	prefixCount        uint32
}

type IPResult struct {
	Country  string
	Province string
	City     string
	Area     string
	Isp      string
}

func New(path string) (*IPSearch, error) {
	p := &IPSearch{}
	//加载ip地址库信息
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p.data = data
	p.prefixMap = make(map[uint32]prefixIndex)

	p.firstStartIpOffset = bytesToLong(data[0], data[1], data[2], data[3])
	p.prefixStartOffset = bytesToLong(data[8], data[9], data[10], data[11])
	p.prefixEndOffset = bytesToLong(data[12], data[13], data[14], data[15])
	p.prefixCount = (p.prefixEndOffset-p.prefixStartOffset)/9 + 1 // 前缀区块每组

	// 初始化前缀对应索引区区间
	indexBuffer := p.data[p.prefixStartOffset:(p.prefixEndOffset + 9)]
	for k := uint32(0); k < p.prefixCount; k++ {
		i := k * 9
		prefix := uint32(indexBuffer[i] & 0xFF)
		pf := prefixIndex{}
		pf.startIndex = bytesToLong(indexBuffer[i+1], indexBuffer[i+2], indexBuffer[i+3], indexBuffer[i+4])
		pf.endIndex = bytesToLong(indexBuffer[i+5], indexBuffer[i+6], indexBuffer[i+7], indexBuffer[i+8])
		p.prefixMap[prefix] = pf
	}
	return p, nil
}

func (p *IPSearch) Get(ip string) (*IPResult, error) {
	ips := strings.Split(ip, ".")
	x, _ := strconv.Atoi(ips[0])
	prefix := uint32(x)
	intIP := ipToLong(ip)

	var high uint32 = 0
	var low uint32 = 0

	if _, ok := p.prefixMap[prefix]; ok {
		low = p.prefixMap[prefix].startIndex
		high = p.prefixMap[prefix].endIndex
	} else {
		return nil, errors.New("no ip data was searched")
	}

	var myIndex uint32
	if low == high {
		myIndex = low
	} else {
		myIndex = p.binarySearch(low, high, intIP)
	}

	ipIndex := ipIndex{}
	ipIndex.getIndex(myIndex, p)

	if ipIndex.startIP <= intIP && ipIndex.endIP >= intIP {
		local := strings.Split(ipIndex.getLocal(p), "|")
		return &IPResult{
			Country:  local[1],
			Province: local[2],
			City:     local[3],
			Area:     local[4],
			Isp:      local[5],
		}, nil
	} else {
		return nil, errors.New("no ip data was searched")
	}
}

// 二分逼近算法
func (p *IPSearch) binarySearch(low uint32, high uint32, k uint32) uint32 {
	var M uint32 = 0
	for low <= high {
		mid := (low + high) / 2

		endIPNum := p.getEndIp(mid)
		if endIPNum >= k {
			M = mid
			if mid == 0 {
				break // 防止溢出
			}
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return M
}

// 只获取结束ip的数值
// 索引区第left个索引
// 返回结束ip的数值
func (p *IPSearch) getEndIp(left uint32) uint32 {
	leftOffset := p.firstStartIpOffset + left*12
	return bytesToLong(p.data[4+leftOffset], p.data[5+leftOffset], p.data[6+leftOffset], p.data[7+leftOffset])
}

func (p *ipIndex) getIndex(left uint32, ips *IPSearch) {
	leftOffset := ips.firstStartIpOffset + left*12
	p.startIP = bytesToLong(ips.data[leftOffset], ips.data[1+leftOffset], ips.data[2+leftOffset], ips.data[3+leftOffset])
	p.endIP = bytesToLong(ips.data[4+leftOffset], ips.data[5+leftOffset], ips.data[6+leftOffset], ips.data[7+leftOffset])
	p.localOffset = bytesToLong3(ips.data[8+leftOffset], ips.data[9+leftOffset], ips.data[10+leftOffset])
	p.localLength = uint32(ips.data[11+leftOffset])
}

// 返回地址信息
// 地址信息的流位置
// 地址信息的流长度
func (p *ipIndex) getLocal(ips *IPSearch) string {
	bytes := ips.data[p.localOffset : p.localOffset+p.localLength]
	return string(bytes)
}

func ipToLong(ip string) uint32 {
	quads := strings.Split(ip, ".")
	l := len(quads)
	var result uint32 = 0
	if l > 3 {
		a, _ := strconv.Atoi(quads[3])
		result += uint32(a)
	}
	if l > 2 {
		b, _ := strconv.Atoi(quads[2])
		result += uint32(b) << 8
	}
	if l > 1 {
		c, _ := strconv.Atoi(quads[1])
		result += uint32(c) << 16
	}
	if l > 0 {
		d, _ := strconv.Atoi(quads[0])
		result += uint32(d) << 24
	}
	return result
}

//字节转整形
func bytesToLong(a, b, c, d byte) uint32 {
	a1 := uint32(a)
	b1 := uint32(b)
	c1 := uint32(c)
	d1 := uint32(d)
	return (a1 & 0xFF) | ((b1 << 8) & 0xFF00) | ((c1 << 16) & 0xFF0000) | ((d1 << 24) & 0xFF000000)
}

func bytesToLong3(a, b, c byte) uint32 {
	a1 := uint32(a)
	b1 := uint32(b)
	c1 := uint32(c)
	return (a1 & 0xFF) | ((b1 << 8) & 0xFF00) | ((c1 << 16) & 0xFF0000)
}
