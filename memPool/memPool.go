package memPool

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sync/atomic"
	"unsafe"
	"github.com/johntech-o/gorpc/utility/convert"
)

func init() {
	fmt.Println("start")
}

const (
	CLASSMASK, CLASSCLEAN Property = 0xF, 0xFFFFFFF0
)

type Property uint32

func (p Property) index() int {
	return int(p & CLASSMASK)
}

func (p *Property) setIndex(index int) {
	*p &= CLASSCLEAN
	*p |= Property(index) & CLASSMASK
}

type stat struct {
	count int64
}

func (s *stat) incre() {
	atomic.AddInt64(&s.count, 1)
}

func (s *stat) decre() {
	atomic.AddInt64(&s.count, -1)
}

func (s *stat) load() int64 {
	return atomic.LoadInt64(&s.count)
}

type MemPool struct {
	baseSize    int
	classAmount int
	bufLists    []unsafe.Pointer
	bufStat     []stat
	missStat    int64
}

func New(baseSize int, classAmount int) *MemPool {
	if classAmount < 1 {
		panic("new MemMool invalid params ")
	}
	mp := MemPool{
		baseSize:    baseSize,
		classAmount: classAmount,
		bufLists:    make([]unsafe.Pointer, classAmount),
		bufStat:     make([]stat, classAmount),
	}
	return &mp
}

// @todo verify size limit
func (mp *MemPool) ChunkSize(index int) int {
	return mp.baseSize * (index + 1)
}

func (mp *MemPool) MaxChunkSize() int {
	return mp.baseSize * mp.classAmount
}

func (mp *MemPool) Malloc(length int) *ElasticBuf {
	if length <= mp.baseSize {
		return mp.popFromList(0)
	}
	index := math.Ceil(float64(length)/float64(mp.baseSize)) - 1
	if length > mp.MaxChunkSize() {
		// println("exceed max chunk size")
		atomic.AddInt64(&mp.missStat, 1)
		return NewElasticBuf(int(index), mp)
	}
	fmt.Println("malloc lenght is ", length, "index is ", index)
	return mp.popFromList(int(index))
}

// @todo caculate miss or hit statistics
func (mp *MemPool) popFromList(index int) *ElasticBuf {
	bufList := &mp.bufLists[index]
	// fmt.Printf("pop bufList %p mp.bufList[index] %p \n", bufList, &mp.bufLists[index])
	var ptr unsafe.Pointer
	for {
		ptr = atomic.LoadPointer(bufList)
		if ptr == nil {
			// fmt.Println("pop ptr is nil index is ", index)
			goto final
		}
		// fmt.Printf("pop load bufList ptr %p", ptr)
		if atomic.CompareAndSwapPointer(bufList, ptr, ((*ElasticBuf)(ptr)).next) {
			// fmt.Println("pop malloc from pool success")
			mp.bufStat[index].decre()
			goto final
		}
	}
final:
	if ptr == nil {
		p := NewElasticBuf(index, mp)
		// fmt.Printf("pop ptr==nil  make a new buf pointer address %p, buf address %p\n", &p, p)
		return p
	}
	return (*ElasticBuf)(ptr)
}

func (mp *MemPool) Free(buf *ElasticBuf) {
	index := buf.Index()
	if index >= mp.classAmount {
		// fmt.Println("memPool free discard index is ", index)
		return
	}
	bufList := &mp.bufLists[index]
	buf.Reset()
	for {
		buf.next = atomic.LoadPointer(bufList)
		if atomic.CompareAndSwapPointer(bufList, buf.next, unsafe.Pointer(buf)) {
			// fmt.Printf("free bufList %p mp.bufList[index] %p  unsafe.Pointer buf %p \n", bufList, &mp.bufLists[index], unsafe.Pointer(buf))
			mp.bufStat[index].incre()
			break
		}
	}
	return
}

func (mp *MemPool) Status(output bool) (s []int64) {
	for _, stat := range mp.bufStat {
		count := stat.load()
		s = append(s, count)
	}
	s = append(s, atomic.LoadInt64(&mp.missStat))
	if output {
		for index, result := range s {
			if index == len(s)-1 {
				fmt.Printf("chunk stat missed %d, count %d\n", index, result)
				continue
			}
			fmt.Printf("chunk stat index  %d, count %d\n", index, result)

		}
	}
	return s
}

type ElasticBuf struct {
	buf      []byte
	writePos int
	property Property
	pp       *MemPool
	next     unsafe.Pointer
}

func (eb *ElasticBuf) Reset() {
	eb.writePos = 0
	eb.next = nil
}

func (eb *ElasticBuf) free() {
	eb.pp.Free(eb)
}

func NewElasticBuf(index int, pool *MemPool) *ElasticBuf {
	buf := ElasticBuf{buf: make([]byte, pool.ChunkSize(index)), pp: pool}
	buf.SetIndex(index)
	return &buf
}

// ElasticBuffer index in pool
func (eb ElasticBuf) Index() int {
	return eb.property.index()
}

func (eb *ElasticBuf) SetIndex(index int) {
	eb.property.setIndex(index)
}

func (eb *ElasticBuf) ReadInt16(reader io.Reader, order convert.ByteOrder) (int16, error) {
	tmpBuf, err := eb.ReadBytes(reader, 2)
	if err != nil {
		return 0, err
	}
	return convert.StreamToInt16(tmpBuf, order), nil
}

func (eb *ElasticBuf) ReadInt32(reader io.Reader, order convert.ByteOrder) (int32, error) {
	tmpBuf, err := eb.ReadBytes(reader, 4)
	if err != nil {
		return 0, err
	}
	return convert.StreamToInt32(tmpBuf, order), nil
}

// the slice returned by this function is valid before next read
func (eb *ElasticBuf) ReadBytes(reader io.Reader, n int) ([]byte, error) {
	tmpBuf := eb.MallocTmpBytes(n)
	c, err := reader.Read(tmpBuf)
	if c != n {
		return nil, errors.New("read fail to buffer")
	}
	if err != nil {
		return nil, err
	}
	return tmpBuf, nil
}

func (eb *ElasticBuf) AppendInt16(v int16, order convert.ByteOrder) error {
	buffer := eb.MallocStreamBytes(2)
	return convert.Int16ToStreamEx(buffer, v, order)
}

func (eb *ElasticBuf) AppendInt32(v int32, order convert.ByteOrder) error {
	buffer := eb.MallocStreamBytes(4)
	return convert.Int32ToStreamEx(buffer, v, order)
}

func (eb *ElasticBuf) FlushToWriter(writer io.Writer) error {
	_, err := writer.Write(eb.buf[0:eb.writePos])
	eb.writePos = 0
	return err
}

// @todo length check and change from global
func (eb *ElasticBuf) MallocStreamBytes(n int) []byte {
	start := eb.writePos
	eb.writePos += n
	return eb.buf[start:eb.writePos]
}

// @todo length check and change from global
func (eb *ElasticBuf) MallocTmpBytes(n int) []byte {
	if cap(eb.buf) < n {
		newBuf := eb.pp.Malloc(n)
		eb.swapWithOther(newBuf)
	}
	return eb.buf[0:n]
}

func (eb *ElasticBuf) swapWithOther(newBuf *ElasticBuf) {
	tmpIndex := newBuf.Index()
	tmpBuf := newBuf.buf
	newBuf.SetIndex(eb.Index())
	newBuf.buf = eb.buf
	eb.SetIndex(tmpIndex)
	eb.buf = tmpBuf
	newBuf.free()
	return
}
