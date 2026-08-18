package ring

import "sort"

// Float32Ring реализует кольцевой буфер значений float32 фиксированного размера.
// При переполнении самые старые значения вытесняются. Значения хранятся в
// порядке добавления (от старого к новому) и могут быть усреднены или
// отфильтрованы по медиане.
type Float32Ring struct {
	buf  []float32
	size int
	head int // index следующей записи
	full bool
}

// NewFloat32Ring создаёт кольцевой буфер на size элементов.
func NewFloat32Ring(size int) *Float32Ring {
	return &Float32Ring{
		buf:  make([]float32, size),
		size: size,
	}
}

// Add помещает значение в буфер, вытесняя самое старое при переполнении.
func (r *Float32Ring) Add(v float32) {
	r.buf[r.head] = v
	r.head = (r.head + 1) % r.size
	if r.head == 0 {
		r.full = true
	}
}

// Len возвращает число заполненных элементов.
func (r *Float32Ring) Len() int {
	if r.full {
		return r.size
	}
	return r.head
}

// Full сообщает, заполнен ли буфер целиком.
func (r *Float32Ring) Full() bool {
	return r.full
}

// Values возвращает срез копий значений в порядке добавления.
func (r *Float32Ring) Values() []float32 {
	n := r.Len()
	out := make([]float32, n)
	start := r.head - n
	if start < 0 {
		start += r.size
	}
	for i := 0; i < n; i++ {
		out[i] = r.buf[(start+i)%r.size]
	}
	return out
}

// Average возвращает среднее арифметическое по заполненным значениям.
// Для пустого буфера возвращает 0 и false.
func (r *Float32Ring) Average() (float32, bool) {
	n := r.Len()
	if n == 0 {
		return 0, false
	}
	vals := r.Values()
	var sum float64
	for _, v := range vals {
		sum += float64(v)
	}
	return float32(sum / float64(n)), true
}

// Median возвращает медиану по заполненным значениям.
// Для пустого буфера возвращает 0 и false.
func (r *Float32Ring) Median() (float32, bool) {
	n := r.Len()
	if n == 0 {
		return 0, false
	}
	vals := r.Values()
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	mid := n / 2
	if n%2 == 1 {
		return vals[mid], true
	}
	return (vals[mid-1] + vals[mid]) / 2, true
}
