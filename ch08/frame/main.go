// main.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var outputFile *os.File

func main() {
	// Открываем файл для записи
	var err error
	outputFile, err = os.OpenFile("frames.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	writeLine("Ethernet-кадры в реальном времени (с фильтром портов)")
	writeLine("Запущено: " + time.Now().Format("2006-01-02 15:04:05"))
	writeLine(strings.Repeat("=", 80))

	// Находим все интерфейсы
	devices, err := pcap.FindAllDevs()
	if err != nil {
		panic(err)
	}

	if len(devices) == 0 {
		fmt.Println("Нет сетевых интерфейсов! Установи Npcap.")
		return
	}

	// Выводим список интерфейсов для выбора
	fmt.Println("\nДоступные интерфейсы:")
	for i, device := range devices {
		fmt.Printf("[%d] %s - %s (IP: %s)\n", i, device.Name, device.Description, getIPs(device))
	}
	fmt.Print("Выбери номер интерфейса (Enter для авто-выбора с IP): ")

	// Читаем выбор интерфейса
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var selectedDevice pcap.Interface
	if input == "" {
		// Авто-выбор: первый с IP
		for _, device := range devices {
			if len(device.Addresses) > 0 {
				selectedDevice = device
				break
			}
		}
		if len(selectedDevice.Name) == 0 {
			selectedDevice = devices[0]
		}
	} else {
		idx, err := strconv.Atoi(input)
		if err != nil || idx < 0 || idx >= len(devices) {
			fmt.Println("Неверный номер! Беру первый.")
			selectedDevice = devices[0]
		} else {
			selectedDevice = devices[idx]
		}
	}

	// Открываем выбранный интерфейс
	handle, err := pcap.OpenLive(selectedDevice.Name, 65536, true, pcap.BlockForever)
	if err != nil {
		panic(fmt.Sprintf("Ошибка открытия %s: %v", selectedDevice.Name, err))
	}
	defer handle.Close()

	fmt.Printf("\nСлушаю интерфейс: %s (%s)\n", selectedDevice.Name, selectedDevice.Description)

	// Выбор фильтра порта
	fmt.Print("\nФильтр по порту (примеры: 'tcp port 443', 'udp port 53', 'icmp'; Enter для всего): ")
	filterInput, _ := reader.ReadString('\n')
	filterInput = strings.TrimSpace(filterInput)
	if filterInput == "" {
		filterInput = "" // Всё
	} else {
		fmt.Printf("Фильтр: '%s' (только эти пакеты)\n", filterInput)
		err = handle.SetBPFFilter(filterInput)
		if err != nil {
			fmt.Printf("Ошибка фильтра: %v. Ловим всё.\n", err)
		}
	}

	fmt.Println("Все кадры сохраняются в frames.txt — открывай его в блокноте!")
	fmt.Println("Генерируй трафик: ping google.com (для ICMP) или открой браузер (для 443).")
	writeLine("Интерфейс: " + selectedDevice.Description)
	if filterInput != "" {
		writeLine("Фильтр: " + filterInput)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	counter := 0
	for packet := range packetSource.Packets() {
		counter++
		printFrame(packet)
		if counter > 100 { // Останавливаем после 100 кадров (убери для бесконечно)
			fmt.Println("\nЗахвачено 100 кадров. Останавливаю. Проверь frames.txt!")
			break
		}
	}
}

func getIPs(device pcap.Interface) string {
	if len(device.Addresses) == 0 {
		return "нет IP"
	}
	return device.Addresses[0].IP.String()
}

func printFrame(packet gopacket.Packet) {
	data := packet.Data() // весь кадр в байтах (включая FCS!)

	writeLine(strings.Repeat("-", 80))
	writeLine(fmt.Sprintf("Время: %s | Длина кадра: %d байт", time.Now().Format("15:04:05.000"), len(data)))

	// Попытка распарсить Ethernet
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		eth := ethLayer.(*layers.Ethernet)

		writeLine(fmt.Sprintf("Destination MAC → %s", eth.DstMAC))
		writeLine(fmt.Sprintf("Source MAC      → %s", eth.SrcMAC))

		// VLAN-тег (если есть)
		if vlan := packet.Layer(layers.LayerTypeDot1Q); vlan != nil {
			v := vlan.(*layers.Dot1Q)
			writeLine(fmt.Sprintf("VLAN ID: %d (приоритет %d)", v.VLANIdentifier, v.Priority))
		}

		writeLine(fmt.Sprintf("Тип: %s (0x%04x)", eth.EthernetType, uint16(eth.EthernetType)))
	} else {
		writeLine("Не Ethernet-кадр (возможно, другой тип)")
	}

	// Красивый hex-дамп всего кадра
	writeLine("Полный кадр в hex (как на проводе):")
	hexDump(data)

	writeLine("")
}

func hexDump(data []byte) {
	const bytesPerLine = 16
	for i := 0; i < len(data); i += bytesPerLine {
		line := data[i:min(i+bytesPerLine, len(data))]

		// Hex часть
		hex := ""
		ascii := ""
		for j, b := range line {
			hex += fmt.Sprintf("%02X ", b)
			if b >= 32 && b <= 126 {
				ascii += string(b)
			} else {
				ascii += "."
			}
			if j == 7 {
				hex += " "
			}
		}

		// Дополняем пробелами короткие строки
		for len(hex) < 48 {
			hex += "   "
		}

		// Подсвечиваем важные части (простая эвристика)
		prefix := "   "
		if i == 0 && len(data) >= 8 && data[0] == 0x55 && data[6] == 0x55 && data[7] == 0xD5 {
			prefix = "ПРМ" // Преамбула (если Npcap её захватил)
		} else if i < 6 {
			prefix = "Dst" // Destination MAC
		} else if i < 12 {
			prefix = "Src" // Source MAC
		} else if i == 12 && len(data) > 16 && data[12] == 0x81 && data[13] == 0x00 {
			prefix = "VLN" // VLAN-тег
		} else if i >= len(data)-4 && i < len(data) {
			prefix = "FCS" // Frame Check Sequence
		}

		writeLine(fmt.Sprintf("%04X  %s %s %s", i, prefix, hex, ascii))
	}
}

func writeLine(s string) {
	fmt.Println(s)
	outputFile.WriteString(s + "\n")
	if err := outputFile.Sync(); err != nil {
		fmt.Printf("Ошибка записи: %v\n", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
