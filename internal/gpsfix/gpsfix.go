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

// Tracker читает NMEA-предложения через готовый драйвер gps.Device (блокирующе,
// в фоновой горутине), парсит их готовым gps.Parser и хранит последний валидный
// фикс вместе с флагом актуальности.
type Tracker struct {
	dev       sentenceSource
	parser    gps.Parser
	lastValid gps.Fix
	hasFix    bool
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
			t.lastValid = fix
			t.hasFix = true
		} else {
			t.hasFix = false
		}
	}
}

// Fix возвращает последний известный валидный фикс (сохраняется и при потере сигнала).
func (t *Tracker) Fix() gps.Fix {
	return t.lastValid
}

// Valid сообщает, есть ли на текущий момент валидный фикс.
func (t *Tracker) Valid() bool {
	return t.hasFix
}

// EverFixed сообщает, был ли получен хотя бы один валидный фикс с момента старта.
func (t *Tracker) EverFixed() bool {
	return t.everFixed
}

// LocalTime возвращает локальное время (UTC + заданный сдвиг).
func (t *Tracker) LocalTime() time.Time {
	return t.lastValid.Time.Add(t.tzOffset)
}
