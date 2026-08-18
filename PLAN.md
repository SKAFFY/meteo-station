# План реализации: анализатор качества воздуха (RP2350 + EPD)

## Цель
Портативный анализатор качества воздуха на TinyGo: SCD30 (CO₂/T/RH) + ENS160 (eCO₂/VOC) на общей I²C‑шине, GPS (NMEA) по UART, вывод на EPD 2.9″ (SSD1680) двумя зонами, кольцевые буферы с фильтрацией.

## MVP-границы
- Сбор данных: SCD30 и ENS160 (без AHT21 — T/RH берём из SCD30).
- GPS: парсинг NMEA, хранение последнего валидного фикса, флаг валидности.
- Кольцевые буферы N=5 (10 с) + усреднение перед выводом.
- Отображение: ч/б на драйвере `epd2in9v2` (SSD1680) — контур готов под добавление красного.
- Поведение при потере GPS: показываются последние известные координаты/время + статус «Waiting for satellite...».
- Пропуск `Display()`, если данные не изменились (экономия ресурса EPD).
- Исключено из MVP: логирование на флеш, кнопки, красный цвет, инверсия.

## Схема пинов (из README)
| Модуль | Пины |
|---|---|
| I²C шина (SCD30 0x61, ENS160 0x53) | I2C0: SDA=GP0, SCL=GP1 (100 кГц) |
| GPS (NMEA 9600) | UART1: RX=GP5 (от GPS TX), TX=GP4 |
| EPD 2.9″ (SSD1680) | SPI1: SCK=GP10, MOSI=GP11, CS=GP12, DC=GP13, RST=GP14, BUSY=GP15 |

## Зависимости (go.mod)
```
tinygo.org/x/drivers v0.35.0            // gps, ens160, waveshare-epd/epd2in9v2
github.com/antonfisher/scd30 v0.4.0     // SCD30 (в drivers пакета нет)
tinygo.org/x/tinyfont                   // текст на EPD
```
Примечание: README ссылается на `tinygo.org/x/drivers/scd30`, но такого пакета не существует — используются драйверы выше.

## Структура
```
cmd/main.go                   — оркестрация: init периферии + бесконечный цикл
internal/ring/ring.go         — кольцевой буфер float32 (Add/Average/Median)
internal/ring/ring_test.go    — unit-тесты
internal/sensors/sensors.go   — интерфейс Sensor + SCD30/ENS160 + снапшот
internal/gpsfix/gpsfix.go     — обёртка gps + последний фикс + флаг валидности
internal/ui/ui.go             — инициализация epd2in9v2 + отрисовка зон А/Б
Makefile                      — MAIN ?= cmd/main.go, TARGET ?= pico2
```

## Главный цикл
- каждые 2 c: SCD30 `HasDataReady`→`ReadMeasurement` → буферы CO₂/T/RH; ENS160 `Update(Concentration)` → буферы eCO₂/TVOC; читать NMEA, `parser.Parse`, обновить фикс.
- каждые 5 c: усреднить (`ring.Average`), нарисовать зоны, `Display()` (с пропуском если не изменилось).

## Обработка ошибок
- Датчик недоступен → значение не добавляется в буфер, на экране `---`.
- Сбой инициализации EPD → мигать `machine.LED`.

## Верификация
- `go test ./...` — unit‑тесты чистой логики.
- `tinygo build -target=pico2 -o build/firmware cmd/main.go` — контроль компиляции.
