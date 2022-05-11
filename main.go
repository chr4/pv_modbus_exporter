package main

import (
	"encoding/binary"
	"github.com/goburrow/modbus"
	"log"
)

func main() {
	handler := modbus.NewTCPClientHandler("localhost:1502")
	handler.SlaveId = 3
	err := handler.Connect()
	defer handler.Close()

	client := modbus.NewClient(handler)

	// Tagesertrag kWh
	results, err := client.ReadInputRegisters(30517, 4)
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}
	dailyYield := binary.BigEndian.Uint64(results)

	// MPPT1
	results, err = client.ReadInputRegisters(30773, 2)
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}
	mppt1Watts := binary.BigEndian.Uint32(results)

	// MPPT2
	results, err = client.ReadInputRegisters(30961, 2)
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}
	mppt2Watts := binary.BigEndian.Uint32(results)

	// Leistung
	results, err = client.ReadInputRegisters(30775, 2)
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}
	totalWatts := binary.BigEndian.Uint32(results)

	log.Printf("north: %dW\n", mppt1Watts)
	log.Printf("south: %dW\n", mppt2Watts)
	log.Printf("total: %dW (daily yield: %dkWh)\n\n", totalWatts, dailyYield)
}
