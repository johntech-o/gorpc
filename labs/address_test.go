package labs

import (
	"fmt"
	"testing"
)

/*
used in timewheel
a      is 0x0, &a        is 0xc8200121a0
s      is 0x0, &s        is 0xc82002c008
a[0]   is 0xc82002c018, &a[0]     is 0xc82008a000
a[1]   is 0xc82002c028, &a[1]     is 0xc82008a008
a[2]   is 0xc82002c030, &a[2]     is 0xc82008a010
a[3]   is 0xc82002c038, &a[3]     is 0xc82008a018
s      is 0xc82002c038, &s        is 0xc82002c008
a[3].C is 0xc82001a180, &(a[3].C) is 0xc82002c038
a[3]   is 0xc82002c070, &a[3]     is 0xc82008a018
s      is 0xc82002c038, &s        is 0xc82002c008
a[3].C is 0xc82001a420, &(a[3].C) is 0xc82002c070
*/
func TestChanAssign(t *testing.T) {
	type Time struct {
		C <-chan struct{}
	}
	var a []*Time
	var s *Time
	fmt.Printf("a      is %p, &a        is %p\n", a, &a)
	fmt.Printf("s      is %p, &s        is %p\n", s, &s)
	for i := 0; i < 10; i++ {
		a = append(a, &Time{C: make(chan struct{}, 1)})
	}
	s = a[3]
	fmt.Printf("a[0]   is %p, &a[0]     is %p\n", a[0], &a[0])
	fmt.Printf("a[1]   is %p, &a[1]     is %p\n", a[1], &a[1])
	fmt.Printf("a[2]   is %p, &a[2]     is %p\n", a[2], &a[2])
	fmt.Printf("a[3]   is %p, &a[3]     is %p\n", a[3], &a[3])
	fmt.Printf("s      is %p, &s        is %p\n", s, &s)
	fmt.Printf("a[3].C is %p, &(a[3].C) is %p\n", a[3].C, &(a[3].C))
	a[3] = &Time{C: make(chan struct{}, 1)}
	fmt.Printf("a[3]   is %p, &a[3]     is %p\n", a[3], &a[3])
	fmt.Printf("s      is %p, &s        is %p\n", s, &s)
	fmt.Printf("a[3].C is %p, &(a[3].C) is %p\n", a[3].C, &(a[3].C))

}

func TestChanClose(t *testing.T) {
	var b <-chan struct{}
	a := make(chan struct{}, 1)
	b = a
	// close(b) // can not be closed
	close(a)
	if _, ok := <-b; !ok {
		t.Log("b has been closed")
	}
	fmt.Println(a, b)
}
