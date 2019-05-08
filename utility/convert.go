package convert

import (
	"errors"
)

type ByteOrder int

const (
	BigEndian ByteOrder = iota
	LittleEndian
)

func StreamToInt64(stream []byte, byteOrder ByteOrder) int64 {
	if len(stream) != 8 {
		return 0
	}
	var u uint64
	if byteOrder == LittleEndian {
		for i := 0; i < 8; i++ {
			u += uint64(stream[i]) << uint(i*8)
		}
	} else {
		for i := 0; i < 8; i++ {
			u += uint64(stream[i]) << uint(8*(7-i))
		}
	}
	return int64(u)
}

func Int64ToStream(i int64, byteOrder ByteOrder) []byte {
	u := uint64(i)
	stream := [8]byte{0, 0, 0, 0, 0, 0, 0, 0}
	if byteOrder == LittleEndian {
		for i := 0; i < 8; i++ {
			stream[i] = byte(u >> uint(8*i))
		}
	} else {
		for i := 0; i < 8; i++ {
			stream[i] = byte(u >> uint(8*(7-i)))
		}
	}
	return stream[:]
}

func Int64ToStreamEx(stream []byte, i int64, byteOrder ByteOrder) error {
	if len(stream) != 8 {
		return errors.New("bad stream")
	}
	u := uint64(i)
	if byteOrder == LittleEndian {
		for i := 0; i < 8; i++ {
			stream[i] = byte(u >> uint(8*i))
		}
	} else {
		for i := 0; i < 8; i++ {
			stream[i] = byte(u >> uint(8*(7-i)))
		}
	}
	return nil
}

func StreamToInt32(stream []byte, byteOrder ByteOrder) int32 {
	if len(stream) != 4 {
		return 0
	}
	var u uint32
	if byteOrder == LittleEndian {
		for i := 0; i < 4; i++ {
			u += uint32(stream[i]) << uint(i*8)
		}
	} else {
		for i := 0; i < 4; i++ {
			u += uint32(stream[i]) << uint(8*(3-i))
		}
	}
	return int32(u)
}

func Int32ToStream(i int32, byteOrder ByteOrder) []byte {
	u := uint32(i)
	stream := [4]byte{0, 0, 0, 0}
	if byteOrder == LittleEndian {
		for i := 0; i < 4; i++ {
			stream[i] = byte(u >> uint(8*i))
		}
	} else {
		for i := 0; i < 4; i++ {
			stream[i] = byte(u >> uint(8*(3-i)))
		}
	}
	return stream[:]
}

func Int32ToStreamEx(stream []byte, i int32, byteOrder ByteOrder) error {
	if len(stream) != 4 {
		return errors.New("bad stream")
	}
	u := uint32(i)
	if byteOrder == LittleEndian {
		for i := 0; i < 4; i++ {
			stream[i] = byte(u >> uint(8*i))
		}
	} else {
		for i := 0; i < 4; i++ {
			stream[i] = byte(u >> uint(8*(3-i)))
		}
	}
	return nil
}

func StreamToInt16(stream []byte, byteOrder ByteOrder) int16 {
	if len(stream) != 2 {
		return 0
	}
	var u uint16
	if byteOrder == LittleEndian {
		for i := 0; i < 2; i++ {
			u += uint16(stream[i]) << uint(i*8)
		}
	} else {
		for i := 0; i < 2; i++ {
			u += uint16(stream[i]) << uint(8*(1-i))
		}
	}
	return int16(u)
}

func Int16ToStream(i int16, byteOrder ByteOrder) []byte {
	u := uint16(i)
	stream := [2]byte{0, 0}
	if byteOrder == LittleEndian {
		for i := 0; i < 2; i++ {
			stream[i] = byte(u >> uint(8*i))
		}
	} else {
		for i := 0; i < 2; i++ {
			stream[i] = byte(u >> uint(8*(1-i)))
		}
	}
	return stream[:]
}

func Int16ToStreamEx(stream []byte, i int16, byteOrder ByteOrder) error {
	if len(stream) != 2 {
		return errors.New("bad stream")
	}
	u := uint16(i)
	if byteOrder == LittleEndian {
		for i := 0; i < 2; i++ {
			stream[i] = byte(u >> uint(8*i))
		}
	} else {
		for i := 0; i < 2; i++ {
			stream[i] = byte(u >> uint(8*(1-i)))
		}
	}
	return nil
}
func StreamToUint16(stream []byte, byteOrder ByteOrder) uint16 {
	if len(stream) != 2 {
		return 0
	}
	var u uint16
	if byteOrder == LittleEndian {
		for i := 0; i < 2; i++ {
			u += uint16(stream[i]) << uint(i*8)
		}
	} else {
		for i := 0; i < 2; i++ {
			u += uint16(stream[i]) << uint(8*(1-i))
		}
	}
	return u
}

func Uint16ToStreamEx(stream []byte, u uint16, byteOrder ByteOrder) error {
	if len(stream) != 2 {
		return errors.New("bad stream")
	}
	if byteOrder == LittleEndian {
		for i := 0; i < 2; i++ {
			stream[i] = byte(u >> uint(8*i))
		}
	} else {
		for i := 0; i < 2; i++ {
			stream[i] = byte(u >> uint(8*(1-i)))
		}
	}
	return nil
}

func StreamToUint32(stream []byte, byteOrder ByteOrder) uint32 {
	if len(stream) != 4 {
		return 0
	}
	var u uint32
	if byteOrder == LittleEndian {
		for i := 0; i < 4; i++ {
			u += uint32(stream[i]) << uint(i*8)
		}
	} else {
		for i := 0; i < 4; i++ {
			u += uint32(stream[i]) << uint(8*(3-i))
		}
	}
	return u
}
func Uint32ToStream(u int32, byteOrder ByteOrder) []byte {
	stream := [4]byte{0, 0, 0, 0}
	if byteOrder == LittleEndian {
		for i := 0; i < 4; i++ {
			stream[i] = byte(u >> uint(8*i))
		}
	} else {
		for i := 0; i < 4; i++ {
			stream[i] = byte(u >> uint(8*(3-i)))
		}
	}
	return stream[:]
}
