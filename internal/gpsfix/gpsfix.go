package gpsfix

import (
	"time"

	"tinygo.org/x/drivers/gps"
)

// byteSource — узкий источник байтов UART, реализуемый machine.UART.
// Выделен отдельно, чтобы трекер можно было тестировать без железа.
type byteSource interface {
	Buffered() int
	ReadByte() (byte, error)
}

// Fix содержит последние известные координаты, время и статус фикса.
type Fix struct {
	Time       time.Time
	Latitude   float32
	Longitude  float32
	Altitude   int32
	Satellites int16
}

// Tracker читает NMEA с UART неблокирующим способом, парсит предложения
// драйвером gps и хранит последний валидный фикс вместе с флагом валидности.
type Tracker struct {
	uart      byteSource
	parser    gps.Parser
	line      []byte
	last      Fix
	valid     bool
	everFixed bool
	tzOffset  time.Duration
}

// New создаёт трекер, читающий с уже сконфигурированного UART.
// tzOffset задаёт сдвиг для локального времени (например, 3*time.Hour).
func New(uart byteSource, tzOffset time.Duration) *Tracker {
	return &Tracker{uart: uart, tzOffset: tzOffset}
}

// Update потребляет накопленные байты UART, собирает NMEA-строки и парсит их.
// Возвращает true, если статус валидности фикса изменился (полезно для пропуска
// перерисовки экрана).
func (t *Tracker) Update() bool {
	prev := t.valid
	current := false

	var b byte
	for t.uart.Buffered() > 0 {
		b, _ = t.uart.ReadByte()
		if b == '\n' {
			if len(t.line) > 0 && t.consume(t.line) {
				current = true
			}
			t.line = t.line[:0]
			continue
		}
		t.line = append(t.line, b)
	}

	t.valid = current
	return t.valid != prev
}

// consume парсит одну NMEA-строку и возвращает true, если получен валидный фикс.
func (t *Tracker) consume(line []byte) bool {
	// NMEA-строки оканчиваются на \r\n — убираем хвостовой \r.
	for len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	fix, err := t.parser.Parse(string(line))
	if err != nil {
		return false
	}
	if !fix.Valid {
		return false
	}
	t.everFixed = true
	t.last = Fix{
		Time:       fix.Time,
		Latitude:   fix.Latitude,
		Longitude:  fix.Longitude,
		Altitude:   fix.Altitude,
		Satellites: fix.Satellites,
	}
	return true
}

// Fix возвращает последний известный фикс.
func (t *Tracker) Fix() Fix {
	return t.last
}

// Valid сообщает, есть ли на текущий момент валидный фикс.
func (t *Tracker) Valid() bool {
	return t.valid
}

// EverFixed сообщает, был ли получен хотя бы один валидный фикс с момента старта.
func (t *Tracker) EverFixed() bool {
	return t.everFixed
}

// LocalTime возвращает локальное время (UTC + заданный сдвиг).
func (t *Tracker) LocalTime() time.Time {
	return t.last.Time.Add(t.tzOffset)
}
