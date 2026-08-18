package main

import (
	"machine"
	"time"
)

const (
	gpsTX = machine.GP8
	gpsRX = machine.GP9
)

func main() {
	time.Sleep(2 * time.Second)
	println("=== UART Raw Dump ===")

	uart := machine.UART1
	err := uart.Configure(machine.UARTConfig{
		BaudRate: 9600,
		TX:       gpsTX,
		RX:       gpsRX,
	})
	if err != nil {
		println("UART error:", err.Error())
		return
	}
	println("UART configured, waiting for data...")

	buf := make([]byte, 128)
	for {
		n, err := uart.Read(buf)
		if err != nil {
			// ошибка чтения
			continue
		}
		if n > 0 {
			// Выводим полученные байты в шестнадцатеричном виде и как строку
			print("RAW HEX: ")
			for i := 0; i < n; i++ {
				print("0x", buf[i], " ")
			}
			println()
			print("RAW STR: ", string(buf[:n]))
			println()
		}
		time.Sleep(100 * time.Millisecond)
	}
}
