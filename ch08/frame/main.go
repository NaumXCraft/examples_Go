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
	// Открываем файл для записи. Используем os.O_TRUNC, чтобы очистить старое содержимое.
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
		fmt.Println("Нет сетевых интерфейсов! Установите Npcap (WinPcap-совместимый драйвер).")
		return
	}

	// Выводим список интерфейсов для выбора
	fmt.Println("\nДоступные интерфейсы:")
	for i, device := range devices {
		fmt.Printf("[%d] %s - %s (IP: %s)\n", i, device.Name, device.Description, getIPs(device))
	}
	fmt.Print("Выбери номер интерфейса (Enter для авто-выбора с IP): ")

	// Читаем выбор интерфейса (улучшено для Windows)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input) // Удаляем пробелы и переводы строки (\r\n)

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

	// Открываем выбранный интерфейс. SnapLen 65536, Promiscuous mode, BlockForever.
	// Примечание: pcap.OpenLive требует административных прав в Windows.
	handle, err := pcap.OpenLive(selectedDevice.Name, 65536, true, pcap.BlockForever)
	if err != nil {
		panic(fmt.Sprintf("Ошибка открытия %s (попробуйте запустить с правами администратора): %v", selectedDevice.Name, err))
	}
	defer handle.Close()

	fmt.Printf("\nСлушаю интерфейс: %s (%s)\n", selectedDevice.Name, selectedDevice.Description)

	// Выбор фильтра порта
	fmt.Print("\nФильтр BPF (примеры: 'tcp port 443', 'udp port 53', 'icmp'; Enter для всего): ")
	filterInput, _ := reader.ReadString('\n')
	filterInput = strings.TrimSpace(filterInput)
	if filterInput == "" {
		filterInput = "" // Ловим всё
	} else {
		fmt.Printf("Применяю фильтр: '%s'\n", filterInput)
		err = handle.SetBPFFilter(filterInput)
		if err != nil {
			fmt.Printf("Ошибка фильтра BPF: %v. Ловим весь трафик.", err)
			filterInput = ""
		}
	}

	fmt.Println("Все кадры сохраняются в frames.txt — открывай его в блокноте!")
	writeLine("Интерфейс: " + selectedDevice.Description)
	if filterInput != "" {
		writeLine("Фильтр BPF: " + filterInput)
	}
	writeLine(strings.Repeat("-", 80))

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	counter := 0
	for packet := range packetSource.Packets() {
		counter++
		printFrame(packet, counter)
		if counter >= 100 { // Убрал > 100 на >= 100
			fmt.Println("\nЗахвачено 100 кадров. Останавливаю. Проверь frames.txt!")
			break
		}
	}
}

// Вспомогательная функция для получения IP-адресов
func getIPs(device pcap.Interface) string {
	if len(device.Addresses) == 0 {
		return "нет IP"
	}
	// Ищем первый IPv4 или IPv6 адрес
	for _, addr := range device.Addresses {
		if addr.IP != nil {
			return addr.IP.String()
		}
	}
	return "нет IP"
}

func printFrame(packet gopacket.Packet, count int) {
	data := packet.Data() // Полный кадр как получен от драйвера (L1/L2, включая FCS если драйвер его оставил)

	writeLine(strings.Repeat("=", 80))
	writeLine(fmt.Sprintf("#%d | Время: %s | Общая Длина: %d байт",
		count, time.Now().Format("15:04:05.000"), len(data)))

	// ----------------------------------------------------
	// L2: Канальный уровень (Ethernet, MAC-адреса, EtherType, VLAN)
	// ----------------------------------------------------
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		eth := ethLayer.(*layers.Ethernet)
		writeLine(fmt.Sprintf(
			"L2 (Ethernet): %s → %s (EtherType: %s 0x%04x)",
			eth.SrcMAC, eth.DstMAC, eth.EthernetType, uint16(eth.EthernetType),
		))

		if vlan := packet.Layer(layers.LayerTypeDot1Q); vlan != nil {
			v := vlan.(*layers.Dot1Q)
			writeLine(fmt.Sprintf("  ↳ VLAN: ID %d (Приоритет %d)",
				v.VLANIdentifier, v.Priority))
		}
	}

	// ----------------------------------------------------
	// L3: Сетевой уровень (IP-адреса, маршрутизация, TTL/HopLimit)
	// ----------------------------------------------------
	if netLayer := packet.NetworkLayer(); netLayer != nil {
		switch netLayer.LayerType() {
		case layers.LayerTypeIPv4:
			ip := netLayer.(*layers.IPv4)
			writeLine(fmt.Sprintf(
				"L3 (IPv4): %s → %s (TTL: %d, Протокол: %s)",
				ip.SrcIP, ip.DstIP, ip.TTL, ip.Protocol,
			))

		case layers.LayerTypeIPv6:
			ip := netLayer.(*layers.IPv6)
			writeLine(fmt.Sprintf(
				"L3 (IPv6): %s → %s (HopLimit: %d, NextHeader: %s)",
				ip.SrcIP, ip.DstIP, ip.HopLimit, ip.NextHeader,
			))

		default:
			writeLine(fmt.Sprintf("L3 (Не IP): %s", netLayer.LayerType()))
		}
	}

	// ----------------------------------------------------
	// L4: Транспортный уровень (TCP/UDP, порты, Seq/Ack, флаги)
	// ----------------------------------------------------
	if transportLayer := packet.TransportLayer(); transportLayer != nil {
		switch transportLayer.LayerType() {

		case layers.LayerTypeTCP:
			tcp := transportLayer.(*layers.TCP)
			flags := getTCPFlags(tcp)
			writeLine(fmt.Sprintf(
				"L4 (TCP): Port %d → %d (Seq: %d, Ack: %d, Flags: %s)",
				tcp.SrcPort, tcp.DstPort, tcp.Seq, tcp.Ack, flags,
			))

		case layers.LayerTypeUDP:
			udp := transportLayer.(*layers.UDP)
			writeLine(fmt.Sprintf(
				"L4 (UDP): Port %d → %d (Length: %d)",
				udp.SrcPort, udp.DstPort, udp.Length,
			))

		default:
			writeLine(fmt.Sprintf("L4: %s", transportLayer.LayerType()))
		}
	}

	// ----------------------------------------------------
	// L5: Прикладной уровень (DNS, HTTP, TLS, прочий payload)
	// ----------------------------------------------------
	if appLayer := packet.ApplicationLayer(); appLayer != nil {

		// Если это DNS (распознаётся как отдельный L5-протокол)
		if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
			dns := dnsLayer.(*layers.DNS)
			if len(dns.Questions) > 0 {
				q := dns.Questions[0]
				writeLine(fmt.Sprintf(
					"L5 (DNS): Запрос: %s (Тип %s)",
					q.Name, q.Type,
				))
			}
		}

		// Payload прикладного уровня (например HTTP, TLS, RAW данные)
		payload := appLayer.Payload()
		if len(payload) > 0 {
			writeLine(fmt.Sprintf("L5 (Payload): Длина: %d байт", len(payload)))
		}
	}

	writeLine(strings.Repeat("-", 80))
	writeLine("Полный кадр в hex (как на проводе):")
	hexDump(data)
	writeLine("")
}

// Улучшенная функция для вывода флагов TCP
func getTCPFlags(tcp *layers.TCP) string {
	var flags []string
	if tcp.FIN {
		flags = append(flags, "FIN")
	}
	if tcp.SYN {
		flags = append(flags, "SYN")
	}
	if tcp.RST {
		flags = append(flags, "RST")
	}
	if tcp.PSH {
		flags = append(flags, "PSH")
	}
	if tcp.ACK {
		flags = append(flags, "ACK")
	}
	if tcp.URG {
		flags = append(flags, "URG")
	}
	if tcp.ECE {
		flags = append(flags, "ECE")
	}
	if tcp.CWR {
		flags = append(flags, "CWR")
	}

	return strings.Join(flags, "|")
}

// Функция для вывода шестнадцатеричного дампа
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
				hex += " " // Дополнительный пробел между 8-м и 9-м байтом
			}
		}

		// Дополняем пробелами короткие строки
		for len(hex) < 49 {
			hex += " "
		}

		// Эвристика префиксов (убрал FCS, так как его редко захватывают)
		prefix := "   "
		if i == 0 {
			prefix = "Dst" // Destination MAC
		} else if i == 6 {
			prefix = "Src" // Source MAC
		} else if i == 12 {
			prefix = "Type" // EtherType
		}

		// offset
		offsetStr := fmt.Sprintf("0x%02X", i)

		writeLine(fmt.Sprintf("%s  %s %s %s", offsetStr, prefix, hex, ascii))
	}
}

// Вспомогательная функция для записи в консоль и файл
func writeLine(s string) {
	fmt.Println(s)
	outputFile.WriteString(s + "\n")
	if err := outputFile.Sync(); err != nil {
		// Ошибка записи: %v
		// Пропускаем вывод ошибки, чтобы не загромождать консоль
	}
}
