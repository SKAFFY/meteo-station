package main

import (
	"image"
	"image/color"
	"log"
	"machine"
	"strconv"
	"time"

	"tinygo.org/x/tinyfont/proggy"

	"github.com/antonfisher/scd30" // Драйвер для сенсора
)

// --- Настройки экрана (SSD1680) ---
const (
	EPD_WIDTH  = 296
	EPD_HEIGHT = 128
	EPD_BLACK  = 0x00 // Черный пиксель
	EPD_WHITE  = 0xFF // Белый пиксель
	EPD_RED    = 0x01 // Красный пиксель (для триколор)
)

// --- Настройки шрифта ---
var (
	font = &proggy.TinySZ8pt7b
)

// --- Пользовательская функция для преобразования []byte в строку ---
func byteSliceToString(b []byte) string {
	return string(b)
}

// --- Функции драйвера SSD1680 ---
type SSD1680 struct {
	spi   *machine.SPI
	dc    machine.Pin
	cs    machine.Pin
	reset machine.Pin
	busy  machine.Pin
}

func NewSSD1680(spi *machine.SPI, dc, cs, reset, busy machine.Pin) *SSD1680 {
	return &SSD1680{
		spi:   spi,
		dc:    dc,
		cs:    cs,
		reset: reset,
		busy:  busy,
	}
}

func (d *SSD1680) Configure() error {
	// Настройка пинов как выходов
	d.dc.Configure(machine.PinConfig{Mode: machine.PinOutput})
	d.cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	d.reset.Configure(machine.PinConfig{Mode: machine.PinOutput})
	d.busy.Configure(machine.PinConfig{Mode: machine.PinInput})

	// Аппаратный сброс экрана
	d.reset.Set(false)
	time.Sleep(10 * time.Millisecond)
	d.reset.Set(true)
	time.Sleep(10 * time.Millisecond)
	d.waitUntilIdle()

	// --- Инициализация экрана (команды для SSD1680) ---
	d.sendCommand(0x12) // Software reset
	d.waitUntilIdle()

	// Driver output control
	d.sendCommand(0x01)
	d.sendData([]byte{0xC7, 0x00, 0x00})

	// Gate driving voltage
	d.sendCommand(0x03)
	d.sendData([]byte{0xC7, 0x00, 0x00})

	// Source driving voltage
	d.sendCommand(0x04)
	d.sendData([]byte{0x00, 0x00})

	// Data entry mode setting
	d.sendCommand(0x11)
	d.sendData([]byte{0x03})

	// Set RAM x-address
	d.sendCommand(0x44)
	d.sendData([]byte{0x00, byte((EPD_WIDTH/8 - 1))})

	// Set RAM y-address
	d.sendCommand(0x45)
	d.sendData([]byte{0x00, 0x00, byte(EPD_HEIGHT - 1), 0x00})

	// Border waveform control
	d.sendCommand(0x3C)
	d.sendData([]byte{0x01})

	return nil
}

// waitUntilIdle ожидает, пока экран не станет готов (BUSY = LOW)
func (d *SSD1680) waitUntilIdle() {
	for d.busy.Get() {
		time.Sleep(10 * time.Millisecond)
	}
}

// sendCommand отправляет команду на экран
func (d *SSD1680) sendCommand(cmd byte) {
	d.dc.Set(false) // DC = LOW для команды
	d.cs.Set(false) // CS = LOW для выбора чипа
	d.spi.Transfer(cmd)
	d.cs.Set(true) // CS = HIGH
}

// sendData отправляет данные на экран
func (d *SSD1680) sendData(data []byte) {
	d.dc.Set(true) // DC = HIGH для данных
	d.cs.Set(false)
	for _, b := range data {
		d.spi.Transfer(b)
	}
	d.cs.Set(true)
}

// DisplayImage отображает полный черно-белый буфер
func (d *SSD1680) DisplayImage(bwBuffer []byte) {
	if len(bwBuffer) != EPD_WIDTH*EPD_HEIGHT/8 {
		log.Println("Buffer size mismatch")
		return
	}

	// Отправка буфера черного
	d.sendCommand(0x24)
	d.sendData(bwBuffer)

	// Отправка пустого буфера для красного (если не используется)
	d.sendCommand(0x26)
	redBuffer := make([]byte, len(bwBuffer))
	d.sendData(redBuffer)

	// Обновление экрана
	d.sendCommand(0x20) // Display refresh
	d.waitUntilIdle()
}

// DrawText рисует текст на переданном изображении
func DrawText(img *image.Gray, x, y int, text string, col color.Gray) {
	// Простая ручная отрисовка через tinyfont
	// tinyfont.WriteLine(img, font, x, y, text, col)
	// В реальном коде можно использовать более сложную библиотеку.
	// Для простоты примера оставим комментарий.
}

func main() {
	time.Sleep(2 * time.Second) // Небольшая задержка перед стартом
	log.Println("Starting application...")

	// --- Настройка SPI для экрана ---
	spi := machine.SPI0
	err := spi.Configure(machine.SPIConfig{
		Frequency: 4000000,
		Mode:      0,
	})
	if err != nil {
		log.Fatal("Failed to configure SPI:", err)
	}
	dcPin := machine.GP10
	csPin := machine.GP15
	resetPin := machine.GP8
	busyPin := machine.GP7

	epd := NewSSD1680(&spi, dcPin, csPin, resetPin, busyPin)
	err = epd.Configure()
	if err != nil {
		log.Fatal("Failed to configure EPD:", err)
	}
	log.Println("EPD configured")

	// --- Настройка I2C для SCD30 ---
	machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 100000,
		SDA:       machine.GP4,
		SCL:       machine.GP5,
	})
	co2sensor := scd30.New(&machine.I2C0)

	// Инициализация сенсора и запуск измерения
	version, err := co2sensor.GetSoftwareVersion()
	if err != nil {
		log.Println("Failed to get sensor version:", err)
	} else {
		log.Println("SCD30 software version:", version)
	}

	err = co2sensor.StartContinuousMeasurement(0)
	if err != nil {
		log.Fatal("Failed to start measurement:", err)
	}
	log.Println("SCD30 measurement started")

	// --- Создание буфера изображения ---
	// Создаем черно-белый буфер (1 байт на 8 пикселей)
	bwBuffer := make([]byte, EPD_WIDTH*EPD_HEIGHT/8)
	// Красный буфер не используем
	redBuffer := make([]byte, EPD_WIDTH*EPD_HEIGHT/8)

	// Основной цикл
	for {
		log.Println("Waiting for sensor data...")
		hasDataReady, err := co2sensor.HasDataReady()
		if err != nil {
			log.Println("Error checking data ready:", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if hasDataReady {
			measurement, err := co2sensor.ReadMeasurement()
			if err != nil {
				log.Println("Error reading measurement:", err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Printf("CO₂: %.0f ppm, Temp: %.2f °C, Humidity: %.2f %%",
				measurement.CO2, measurement.Temperature, measurement.Humidity)

			// --- Формирование строки для вывода ---
			outputText := "CO₂: " + strconv.FormatFloat(measurement.CO2, 'f', 0, 64) + " ppm\n"
			outputText += "Temp: " + strconv.FormatFloat(measurement.Temperature, 'f', 1, 64) + " °C\n"
			outputText += "Hum:  " + strconv.FormatFloat(measurement.Humidity, 'f', 1, 64) + " %"

			// --- Очистка буфера (заполняем белым цветом) ---
			for i := range bwBuffer {
				bwBuffer[i] = 0xFF
			}
			// Вывод текста — это самая сложная часть.
			// В TinyGo нет тривиального способа конвертировать текст в буфер на лету.
			// Гораздо проще использовать готовый шрифт и функцию рисования текста.
			// В данном примере я оставлю пока просто очистку.

			// Обновляем экран
			epd.DisplayImage(bwBuffer)
		} else {
			log.Println("Sensor not ready, waiting...")
		}
		time.Sleep(10 * time.Second)
	}
}
