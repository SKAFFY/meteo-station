package main

import (
	"fmt"
	"machine"
	"time"

	"github.com/SKAFFY/meteo-station/internal/gpsfix"
	"github.com/SKAFFY/meteo-station/internal/ring"
	"github.com/SKAFFY/meteo-station/internal/sensors"
	"github.com/SKAFFY/meteo-station/internal/ui"
)

const (
	// I2C шина (SCD30 + ENS160)
	i2cSDA = machine.GP0
	i2cSCL = machine.GP1

	// GPS (NMEA 9600)
	gpsTX = machine.GP4
	gpsRX = machine.GP5

	// EPD 2.9" (SSD1680)
	epdSCLK = machine.GP10
	epdMOSI = machine.GP11
	epdCS   = machine.GP12
	epdDC   = machine.GP13
	epdRST  = machine.GP14
	epdBUSY = machine.GP15
	// SDI (MISO) SPI1 — не используется дисплеем, но обязателен для validPins
	// на RP2350. Никуда физически не подключается.
	epdMISO = machine.GP8

	// тайминги
	pollInterval = 2 * time.Second
	drawInterval = 5 * time.Second
	settleDelay  = 2 * time.Second

	tzOffset = 3 * time.Hour
)

func main() {
	// Дать железу и датчикам стабилизироваться после подачи питания.
	time.Sleep(settleDelay)

	if err := machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 100_000,
		SDA:       i2cSDA,
		SCL:       i2cSCL,
	}); err != nil {
		failBlink(err)
	}

	sens := sensors.New(machine.I2C0)
	// Датчики не критичны: недоступный просто показывается как offline.
	_ = sens.Configure()

	if err := machine.UART1.Configure(machine.UARTConfig{
		BaudRate: 9600,
		TX:       gpsTX,
		RX:       gpsRX,
	}); err != nil {
		failBlink(err)
	}
	tracker := gpsfix.New(machine.UART1, tzOffset)

	if err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 4_000_000,
		SCK:       epdSCLK,
		SDO:       epdMOSI,
		SDI:       epdMISO,
	}); err != nil {
		failBlink(err)
	}
	disp := ui.New(machine.SPI1, epdCS, epdDC, epdRST, epdBUSY)
	disp.Init()

	scdPresent := sens.SCDPresent()
	ensPresent := sens.ENSPresent()

	co2 := ring.NewFloat32Ring(5)
	eco2 := ring.NewFloat32Ring(5)
	tvoc := ring.NewFloat32Ring(5)
	temp := ring.NewFloat32Ring(5)
	rh := ring.NewFloat32Ring(5)

	var lastTemp, lastRH float32
	hasEnv := false

	lastPoll := time.Now()
	lastDraw := time.Now()
	var lastKey string

	for {
		now := time.Now()

		if now.Sub(lastPoll) >= pollInterval {
			lastPoll = now

			// Компенсация ENS160 последними T/RH из SCD30.
			if hasEnv {
				_ = sens.ApplyCompensation(int32(lastTemp*1000), int32(lastRH*1000))
			}

			r := sens.Read()
			if r.SCDOK {
				co2.Add(r.CO2)
				temp.Add(r.Temp)
				rh.Add(r.RH)
				lastTemp = r.Temp
				lastRH = r.RH
				hasEnv = true
			}
			if r.ENSOK {
				eco2.Add(r.Eco2)
				tvoc.Add(r.TVOC)
			}

			tracker.Update()
		}

		if now.Sub(lastDraw) >= drawInterval {
			lastDraw = now
			render(disp, co2, eco2, tvoc, temp, rh, tracker, scdPresent, ensPresent, &lastKey)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func render(d *ui.Display, co2, eco2, tvoc, temp, rh *ring.Float32Ring,
	tracker *gpsfix.Tracker, scdPresent, ensPresent bool, lastKey *string) {

	var v ui.Values

	if avg, ok := co2.Average(); ok {
		v.CO2, v.HasCO2 = avg, true
	}
	if avg, ok := eco2.Average(); ok {
		v.Eco2, v.HasEco2 = avg, true
	}
	if avg, ok := tvoc.Average(); ok {
		v.TVOC, v.HasTVOC = avg, true
	}
	if avg, ok := temp.Average(); ok {
		v.Temp, v.HasTemp = avg, true
	}
	if avg, ok := rh.Average(); ok {
		v.RH, v.HasRH = avg, true
	}

	fix := tracker.Fix()
	v.FixValid = tracker.Valid()
	v.EverFixed = tracker.EverFixed()
	v.SCDPresent = scdPresent
	v.ENSPresent = ensPresent
	v.Latitude = fix.Latitude
	v.Longitude = fix.Longitude
	v.FixTimeStr = fix.Time.Format("15:04:05")
	v.LocalTimeStr = tracker.LocalTime().Format("15:04:05")

	key := frameKey(v)
	if *lastKey == key {
		return
	}
	*lastKey = key

	d.Render(v)
}

// frameKey строит строку-сигнатуру кадра для пропуска перерисовки,
// если данные и статус GPS не изменились.
func frameKey(v ui.Values) string {
	return fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v|%v|%v",
		toInt(v.CO2), toInt(v.Eco2), toInt(v.TVOC), int(v.Temp*10), int(v.RH*10),
		v.HasCO2, v.HasEco2, v.HasTVOC, v.HasTemp, v.HasRH) +
		fmt.Sprintf("|%v|%v|%v|%v|%s|%s|%v|%v",
			v.SCDPresent, v.ENSPresent, v.FixValid, v.EverFixed,
			v.FixTimeStr, v.LocalTimeStr,
			int(v.Latitude*1e4), int(v.Longitude*1e4))
}

func toInt(f float32) int {
	if f < 0 {
		return int(f - 0.5)
	}
	return int(f + 0.5)
}

// failBlink мигает встроенным светодиодом при фатальной ошибке инициализации.
func failBlink(err error) {
	led := machine.LED
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})
	for {
		println(err.Error())
		led.High()
		time.Sleep(200 * time.Millisecond)
		led.Low()
		time.Sleep(200 * time.Millisecond)
	}
}
