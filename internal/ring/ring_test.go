package ring

import (
	"math"
	"testing"
)

func TestAddAndLen(t *testing.T) {
	r := NewFloat32Ring(3)
	if r.Len() != 0 {
		t.Fatalf("ожидался пустой буфер, Len=%d", r.Len())
	}
	r.Add(1)
	r.Add(2)
	if r.Len() != 2 {
		t.Fatalf("ожидался Len=2, получено %d", r.Len())
	}
	if r.Full() {
		t.Fatal("буфер не должен быть полным")
	}
	r.Add(3)
	if !r.Full() {
		t.Fatal("буфер должен быть полным")
	}
}

func TestOverflow(t *testing.T) {
	r := NewFloat32Ring(3)
	for _, v := range []float32{1, 2, 3, 4, 5} {
		r.Add(v)
	}
	vals := r.Values()
	want := []float32{3, 4, 5}
	if len(vals) != len(want) {
		t.Fatalf("ожидалось %d значений, получено %d", len(want), len(vals))
	}
	for i := range want {
		if vals[i] != want[i] {
			t.Fatalf("Values[%d]=%v, ожидалось %v", i, vals[i], want[i])
		}
	}
}

func TestAverage(t *testing.T) {
	r := NewFloat32Ring(3)
	if _, ok := r.Average(); ok {
		t.Fatal("для пустого буфера Average не должен быть доступен")
	}
	r.Add(10)
	r.Add(20)
	r.Add(30)
	avg, ok := r.Average()
	if !ok {
		t.Fatal("ожидался доступный Average")
	}
	if math.Abs(float64(avg)-20) > 1e-6 {
		t.Fatalf("Avg(10,20,30)=%v, ожидалось 20", avg)
	}
}

func TestMedianOdd(t *testing.T) {
	r := NewFloat32Ring(5)
	for _, v := range []float32{5, 1, 3, 2, 4} {
		r.Add(v)
	}
	med, ok := r.Median()
	if !ok {
		t.Fatal("ожидался доступный Median")
	}
	if med != 3 {
		t.Fatalf("Median={5,1,3,2,4}=%v, ожидалось 3", med)
	}
}

func TestMedianEven(t *testing.T) {
	r := NewFloat32Ring(4)
	for _, v := range []float32{40, 10, 30, 20} {
		r.Add(v)
	}
	med, _ := r.Median()
	if med != 25 {
		t.Fatalf("Median={40,10,30,20}=%v, ожидалось 25", med)
	}
}
