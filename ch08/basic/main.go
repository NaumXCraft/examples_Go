package main

// определение интерфейсов на компьютере

import (
	"fmt"

	"github.com/google/gopacket/pcap"
)

func main() {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Список интерфейсов Npcap ===")
	for _, d := range devices {
		fmt.Printf("\nName: %s\n", d.Name)
		fmt.Printf("Desc: %s\n", d.Description)
		fmt.Println("Addresses:")
		for _, a := range d.Addresses {
			fmt.Printf("   IP: %v\n", a.IP)
		}
	}
}
