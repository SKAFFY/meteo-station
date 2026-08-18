package sensors

import (
	"machine"

	"github.com/antonfisher/scd30"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ens160"
)

// Reading содержит текущие показания обоих датчиков за один цикл опроса.
// Флаги SCDOK/ENSOK показывают, какой из датчиков реально дал данные.
type Reading struct {
	CO2   float32
	Eco2  float32
	TVOC  float32
	Temp  float32
	RH    float32
	SCDOK bool
	ENSOK bool
}

// Device инкапсулирует SCD30 (CO2/T/RH) и ENS160 (eCO2/VOC) на общей I2C-шине.
// Поля scdPresent/ensPresent фиксируют, был ли датчик найден при инициализации.
type Device struct {
	scd        scd30.Device
	ens        *ens160.Device
	scdPresent bool
	ensPresent bool
}

// New создаёт обёртку над датчиками на заданной шине.
func New(bus *machine.I2C) *Device {
	return &Device{
		scd: scd30.New(bus),
		ens: ens160.New(bus, 0),
	}
}

// Configure пытается инициализировать каждый датчик независимо.
// Отсутствующий датчик не считается ошибкой: он помечается недоступным,
// а устройство продолжает работать с теми датчиками, которые ответили.
// Всегда возвращает nil — фатальность на уровне датчиков отсутствует.
func (d *Device) Configure() error {
	// SCD30: самоинициализация, при успехе — непрерывный режим с интервалом 2 с.
	if err := d.scd.SetSelfCalibration(true); err == nil {
		if err := d.scd.StartContinuousMeasurement(0); err == nil {
			d.scdPresent = true
		}
	}

	// ENS160: если датчик ответил — переводим в стандартный режим.
	if d.ens.Connected() {
		if err := d.ens.Configure(); err == nil {
			d.ensPresent = true
		}
	}

	return nil
}

// SCDPresent сообщает, был ли SCD30 обнаружен при инициализации.
func (d *Device) SCDPresent() bool { return d.scdPresent }

// ENSPresent сообщает, был ли ENS160 обнаружен при инициализации.
func (d *Device) ENSPresent() bool { return d.ensPresent }

// ApplyCompensation передаёт в ENS160 текущие температуру и влажность
// (в милли-единицах) из SCD30 для компенсации показаний.
func (d *Device) ApplyCompensation(tempMilliC, rhMilliPct int32) error {
	return d.ens.SetEnvDataMilli(tempMilliC, rhMilliPct)
}

// Read опрашивает оба датчика и возвращает объединённое показание.
// Датчик, не найденный при инициализации, не опрашивается.
func (d *Device) Read() Reading {
	var out Reading
	if d.scdPresent {
		if ready, err := d.scd.HasDataReady(); err == nil && ready {
			if m, err := d.scd.ReadMeasurement(); err == nil {
				out.CO2 = m.CO2
				out.Temp = m.Temperature
				out.RH = m.Humidity
				out.SCDOK = true
			}
		}
	}
	if d.ensPresent {
		if err := d.ens.Update(drivers.Concentration); err == nil {
			out.Eco2 = float32(d.ens.ECO2())
			out.TVOC = float32(d.ens.TVOC())
			out.ENSOK = true
		}
	}
	return out
}
