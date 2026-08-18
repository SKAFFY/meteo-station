package ui

import (
	"fmt"
	"image/color"
	"machine"

	"tinygo.org/x/drivers/waveshare-epd/epd2in9v2"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
)

var black = color.RGBA{0, 0, 0, 255}

// Values — данные одного кадра для отрисовки. Булевы флаги разрешают вывод
// "---" для недоступных датчиков.
type Values struct {
	CO2, Eco2, TVOC     float32
	Temp, RH            float32
	HasCO2              bool
	HasEco2, HasTVOC    bool
	HasTemp, HasRH      bool
	SCDPresent          bool
	ENSPresent          bool
	FixValid            bool
	EverFixed           bool
	FixTimeStr          string
	LocalTimeStr        string
	Latitude, Longitude float32
}

// Display — обёртка над EPD 2.9" (SSD1680) с двумя смысловыми зонами.
type Display struct {
	dev  epd2in9v2.Device
	font *tinyfont.Font
}

// New создаёт дисплей на заданной шине SPI и контактах управления.
func New(spi *machine.SPI, cs, dc, rst, busy machine.Pin) *Display {
	return &Display{
		dev:  epd2in9v2.New(spi, cs, dc, rst, busy),
		font: &freesans.Regular9pt7b,
	}
}

// Init настраивает дисплей и очищает его.
func (d *Display) Init() {
	d.dev.Configure(epd2in9v2.Config{})
	d.dev.SetRotation(epd2in9v2.ROTATION_90)
	d.dev.ClearDisplay()
}

// Clear стирает экран (белый фон).
func (d *Display) Clear() {
	d.dev.ClearDisplay()
}

func fmt1(v float32) string {
	if v >= 1000 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

// Render рисует обе зоны и выводит кадр на экран.
func (d *Display) Render(v Values) {
	d.dev.ClearBuffer()

	x := int16(4)
	y := int16(10)
	step := int16(13)

	if v.SCDPresent {
		d.line(x, y, "CO2   "+valOrDashes(v.HasCO2, fmt1(v.CO2))+" ppm")
		y += step
		d.line(x, y, "Temp  "+valOrDashes(v.HasTemp, fmt1(v.Temp))+" C")
		y += step
		d.line(x, y, "RH    "+valOrDashes(v.HasRH, fmt1(v.RH))+" %")
		y += step
	} else {
		d.line(x, y, "SCD30 offline")
		y += step
	}

	if v.ENSPresent {
		d.line(x, y, "eCO2  "+valOrDashes(v.HasEco2, fmt1(v.Eco2))+" ppm")
		y += step
		d.line(x, y, "TVOC  "+valOrDashes(v.HasTVOC, fmt1(v.TVOC))+" ppb")
		y += step
	} else {
		d.line(x, y, "ENS160 offline")
		y += step
	}

	y += 6
	d.line(x, y, "=== GPS ===")
	y += step

	if v.EverFixed {
		d.line(x, y, "UTC "+v.FixTimeStr)
		y += step
		d.line(x, y, "LOC "+v.LocalTimeStr)
		y += step
		if v.Latitude != 0 || v.Longitude != 0 {
			d.line(x, y, fmt.Sprintf("LAT %.6f", v.Latitude))
			y += step
			d.line(x, y, fmt.Sprintf("LNG %.6f", v.Longitude))
			y += step
		}
	} else {
		d.line(x, y, "Waiting for")
		y += step
		d.line(x, y, "satellite...")
	}

	if v.EverFixed && !v.FixValid {
		d.line(x, y+6, "Signal lost")
	}

	_ = d.dev.Display()
}

func valOrDashes(has bool, s string) string {
	if !has {
		return "---"
	}
	return s
}

func (d *Display) line(x, y int16, s string) {
	tinyfont.WriteLine(&d.dev, d.font, x, y, s, black)
}
