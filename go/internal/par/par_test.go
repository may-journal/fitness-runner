package par

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestMapPreservesOrder(t *testing.T) {
	got := Map(100, 8, func(i int) int {
		time.Sleep(time.Duration(rand.Intn(3)) * time.Millisecond)
		return i * 2
	})
	for i, v := range got {
		if v != i*2 {
			t.Fatalf("index %d: got %d", i, v)
		}
	}
}

func TestMapZeroItems(t *testing.T) {
	if got := Map(0, 4, func(i int) int { return i }); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestMapSingleWorkerIsSequentialOrder(t *testing.T) {
	var seen []int
	got := Map(5, 1, func(i int) int {
		seen = append(seen, i)
		return i
	})
	if !reflect.DeepEqual(seen, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("single worker must process in order: %v", seen)
	}
	if !reflect.DeepEqual(got, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("results: %v", got)
	}
}

func TestMapDefaultsWorkers(t *testing.T) {
	got := Map(10, 0, func(i int) int { return i })
	if len(got) != 10 || got[9] != 9 {
		t.Fatalf("got %v", got)
	}
}
