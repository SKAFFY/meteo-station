package gpsfix

import (
	"io"
	"testing"
	"time"
)

// mockSrc эмулирует UART-источник байтов без железа.
type mockSrc struct {
	data []byte
}

func (m *mockSrc) Buffered() int { return len(m.data) }

func (m *mockSrc) ReadByte() (byte, error) {
	if len(m.data) == 0 {
		return 0, io.EOF
	}
	b := m.data[0]
	m.data = m.data[1:]
	return b, nil
}

const validRMC = "$GPRMC,203522.00,A,5109.0262308,N,11401.8407342,W,0.004,133.4,010622,0.0,E,D*2B\r\n"

func TestValidFix(t *testing.T) {
	m := &mockSrc{data: []byte(validRMC)}
	tr := New(m, 3*time.Hour)

	if changed := tr.Update(); !changed {
		t.Fatal("ожидалось изменение статуса валидности")
	}
	if !tr.Valid() {
		t.Fatal("ожидался валидный фикс")
	}
	if !tr.EverFixed() {
		t.Fatal("ожидался флаг EverFixed")
	}

	fix := tr.Fix()
	if fix.Latitude < 51.1 || fix.Latitude > 51.2 {
		t.Fatalf("неожиданная широта: %v", fix.Latitude)
	}
	if fix.Longitude < -114.1 || fix.Longitude > -114.0 {
		t.Fatalf("неожиданная долгота: %v", fix.Longitude)
	}
	if fix.Time.Hour() != 20 || fix.Time.Minute() != 35 || fix.Time.Second() != 22 {
		t.Fatalf("неожиданное время фикса: %v", fix.Time)
	}
	// локальное время = UTC + 3 часа
	if lt := tr.LocalTime(); lt.Hour() != 23 {
		t.Fatalf("ожидалось локальное время 23 ч, получено %v", lt)
	}
}

func TestNoDataNotValid(t *testing.T) {
	m := &mockSrc{}
	tr := New(m, 3*time.Hour)
	tr.Update()
	if tr.Valid() {
		t.Fatal("без данных фикс не должен быть валидным")
	}
	if tr.EverFixed() {
		t.Fatal("без данных фикс не мог быть получен")
	}
}

func TestLossKeepsLastFix(t *testing.T) {
	m := &mockSrc{data: []byte(validRMC)}
	tr := New(m, 3*time.Hour)
	tr.Update()

	// Потеря сигнала: RMC со статусом V (void).
	m.data = append(m.data, []byte("$GPRMC,203600.00,V,5109.0262308,N,11401.8407342,W,0.004,133.4,010622,0.0,E,D*??\r\n")...)
	tr.Update()

	if tr.Valid() {
		t.Fatal("после потери сигнала фикс не должен быть валидным")
	}
	if !tr.EverFixed() {
		t.Fatal("флаг EverFixed должен сохраниться")
	}
	// Последние координаты должны сохраниться.
	if fix := tr.Fix(); fix.Latitude < 51.1 || fix.Latitude > 51.2 {
		t.Fatalf("последняя широта должна сохраниться, получено %v", fix.Latitude)
	}
}
