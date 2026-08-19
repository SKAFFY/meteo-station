package gpsfix

import (
	"io"
	"testing"
	"time"
)

// mockSrc эмулирует источник NMEA-предложений через канал.
// NextSentence блокирует, ожидая новое предложение (как драйвер gps.Device).
type mockSrc struct {
	in chan string
}

func newMock() *mockSrc {
	return &mockSrc{in: make(chan string)}
}

func (m *mockSrc) NextSentence() (string, error) {
	s, ok := <-m.in
	if !ok {
		return "", io.EOF
	}
	return s, nil
}

const validRMC = "$GPRMC,203522.00,A,5109.0262308,N,11401.8407342,W,0.004,133.4,010622,0.0,E,D*2B"
const voidRMC = "$GPRMC,203600.00,V,5109.0262308,N,11401.8407342,W,0.004,133.4,010622,0.0,E,A*2B"

// waitFor ждёт, пока cond вернёт true, с тайм-аутом (чтение идёт в горутине).
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("таймаут ожидания условия")
}

func TestValidFix(t *testing.T) {
	m := newMock()
	tr := New(m, 3*time.Hour)
	tr.Start()
	m.in <- validRMC

	waitFor(t, tr.Valid)
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
	m := newMock()
	tr := New(m, 3*time.Hour)
	tr.Start()

	// Ни одного предложения — фикс не появляется.
	time.Sleep(50 * time.Millisecond)
	if tr.Valid() {
		t.Fatal("без данных фикс не должен быть валидным")
	}
	if tr.EverFixed() {
		t.Fatal("без данных фикс не мог быть получен")
	}
}

func TestLossKeepsLastFix(t *testing.T) {
	m := newMock()
	tr := New(m, 3*time.Hour)
	tr.Start()
	m.in <- validRMC

	waitFor(t, tr.Valid)

	// Потеря сигнала: RMC со статусом V (void).
	m.in <- voidRMC
	waitFor(t, func() bool { return !tr.Valid() })

	if !tr.EverFixed() {
		t.Fatal("флаг EverFixed должен сохраниться")
	}
	// Последние координаты должны сохраниться.
	if fix := tr.Fix(); fix.Latitude < 51.1 || fix.Latitude > 51.2 {
		t.Fatalf("последняя широта должна сохраниться, получено %v", fix.Latitude)
	}
}
