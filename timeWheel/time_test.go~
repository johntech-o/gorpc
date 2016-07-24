package timeWheel

import (
	"fmt"
	"testing"
	"time"
)

func TestTimerCreate(t *testing.T) {
	begin := time.Now()
	t1 := NewTimer(time.Second)
	select {
	case <-t1.C:
		t := time.Now().Sub(begin)
		fmt.Println(t)
	}
}
