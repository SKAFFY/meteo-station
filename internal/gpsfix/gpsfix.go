package gpsfix

import (
	"time"

	"tinygo.org/x/drivers/gps"
)

// sentenceSource — источник готовых NMEA-предложений.
// Реализуется *gps.Device через NextSentence. Выделен, чтобы трекер
// можно было тестировать без железа.
type sentenceSource interface {
	NextSentence() (string, error)
}

// Fix содержит последние известные координаты, время и статус фикса.
type Fix struct {
	Time       time.Time
	Latitude   float32
	Longitude  float32
	Altitude   int32
	Satellites int16
}

// Tracker читает NMEA-предложения через готовый драйвер gps.Device (блокирующе,
// в фоновой горутине), парсит их готовым gps.Parser и хранит последний валидный
// фикс вместе с флагом валидности.
type Tracker struct {
	dev       sentenceSource
	parser    gps.Parser
	last      Fix
	valid     bool
	everFixed bool
	tzOffset  time.Duration
}

// New создаёт трекер на основе сконфигурированного GPS-устройства.
// tzOffset задаёт сдвиг для локального времени (например, 3*time.Hour).
func New(dev sentenceSource, tzOffset time.Duration) *Tracker {
	return &Tracker{dev: dev, tzOffset: tzOffset}
}

// Start запускает фоновую горутину чтения и парсинга GPS.
func (t *Tracker) Start() {
	go t.run()
}

// run блокирующе читает предложения с устройства и обновляет состояние фикса.
func (t *Tracker) run() {
	for {
		sentence, err := t.dev.NextSentence()
		if err != nil {
			continue
		}
		fix, err := t.parser.Parse(sentence)
		if err != nil {
			continue
		}
		if fix.Valid {
			t.everFixed = true
			t.last = Fix{
				Time:       fix.Time,
				Latitude:   fix.Latitude,
				Longitude:  fix.Longitude,
				Altitude:   fix.Altitude,
				Satellites: fix.Satellites,
			}
			t.valid = true
		} else {
			t.valid = false
		}
	}
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
