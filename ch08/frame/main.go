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
	// Инициализация: файл, заголовок, поиск устройств.
	if err := initOutputFile(); err != nil {
		panic(err)
	}
	defer outputFile.Close()

	devices, err := findDevices()
	if err != nil {
		panic(err)
	}
	if len(devices) == 0 {
		fmt.Println("Нет сетевых интерфейсов! Установите Npcap (WinPcap-совместимый драйвер).")
		return
	}

	// Выбор интерфейса и фильтра (модульно, через функции).
	reader := bufio.NewReader(os.Stdin)
	selectedDevice := selectDevice(reader, devices)
	bpfFilter := selectBPFFilter(reader)

	// Открытие handle и применение фильтра.
	handle, err := openInterface(selectedDevice)
	if err != nil {
		panic(err)
	}
	defer handle.Close()

	fmt.Printf("\nСлушаю интерфейс: %s (%s)\n", selectedDevice.Name, selectedDevice.Description)
	if bpfFilter != "" {
		if err := handle.SetBPFFilter(bpfFilter); err != nil {
			fmt.Printf("Ошибка фильтра BPF: %v. Ловим весь трафик.\n", err)
			bpfFilter = ""
		} else {
			fmt.Printf("Применён фильтр: '%s'\n", bpfFilter)
		}
	}

	fmt.Println("Все кадры сохраняются в frames.txt — открывай его в блокноте!")

	writeLine("Интерфейс: " + selectedDevice.Description)
	if bpfFilter != "" {
		writeLine("Фильтр BPF: " + bpfFilter)
	}
	writeLine(strings.Repeat("-", 80))

	// Захват пакетов.
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	counter := 0
	for packet := range packetSource.Packets() {
		counter++
		printFrame(packet, counter)
		if counter >= 100 {
			fmt.Println("\nЗахвачено 100 кадров. Останавливаю. Проверь frames.txt!")
			break
		}
	}
}

// initOutputFile: Инициализация файла лога и вывод заголовка.
// Возвращает error для обработки в main.
func initOutputFile() error {
	var err error
	outputFile, err = os.OpenFile("frames.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	writeLine("Ethernet-кадры в реальном времени (с фильтром портов)")
	writeLine("Запущено: " + time.Now().Format("2006-01-02 15:04:05"))
	writeLine(strings.Repeat("=", 80))
	return nil
}

// findDevices: Поиск всех сетевых интерфейсов.
// Возвращает список и error (для обработки в main).
func findDevices() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// selectDevice: Интерактивный выбор сетевого интерфейса.
// Принимает reader и список устройств, возвращает выбранный.
// Авто-выбор: первый с IP, fallback — первый в списке.
func selectDevice(reader *bufio.Reader, devices []pcap.Interface) pcap.Interface {
	fmt.Println("\nДоступные интерфейсы:")
	for i, device := range devices {
		fmt.Printf("[%d] %s - %s (IP: %s)\n", i, device.Name, device.Description, getIPs(device))
	}
	fmt.Print("Выбери номер интерфейса (Enter для авто-выбора с IP): ")

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
	return selectedDevice
}

// openInterface: Открытие выбранного интерфейса для захвата.
// SnapLen 65536, Promiscuous mode, BlockForever.
func openInterface(device pcap.Interface) (*pcap.Handle, error) {
	return pcap.OpenLive(device.Name, 65536, true, pcap.BlockForever)
}

// selectBPFFilter: Выбор BPF-фильтра через меню (как раньше).
// Принимает reader, возвращает string фильтра.
func selectBPFFilter(reader *bufio.Reader) string {
	fmt.Println("\nВыбери тип фильтра (BPF):")
	fmt.Println("[0] Всё (без фильтра)")
	fmt.Println("[1] TCP/UDP по портам (пример: tcp port 80 or udp port 53)")
	fmt.Println("[2] ICMP (ping)")
	fmt.Println("[3] Служебные L2-протоколы (ARP, LLDP, CDP, GARP, GVRP, MACsec)")
	fmt.Println("[4] Только ARP*")
	fmt.Println("[5] Только LLDP")
	fmt.Println("[6] Только CDP")
	fmt.Println("[7] Только GARP/GVRP")
	fmt.Println("[8] Только MACsec")
	fmt.Println("[9] DNS-запросы (udp port 53)")
	fmt.Print("Или введи свой фильтр (Enter для всего): ")

	filterInput, _ := reader.ReadString('\n')
	filterInput = strings.TrimSpace(filterInput)

	var bpfFilter string
	switch filterInput {
	case "0", "":
		bpfFilter = ""
	case "1":
		bpfFilter = "tcp or udp"
		fmt.Println("Фильтр: tcp or udp (добавь порты вручную, если нужно)")
	case "2":
		bpfFilter = "icmp"
	case "3":
		bpfFilter = "arp or ether proto 0x88cc or ether proto 0x2000 or ether proto 0x886d or stp or ether proto 0x88e5"
		fmt.Println("Фильтр: ARP | LLDP | CDP | GARP/GVRP | MACsec")
	case "4":
		bpfFilter = "arp"
	case "5":
		bpfFilter = "ether proto 0x88cc"
	case "6":
		bpfFilter = "ether proto 0x2000"
	case "7":
		bpfFilter = "ether proto 0x886d or stp"
	case "8":
		bpfFilter = "ether proto 0x88e5"
	case "9":
		bpfFilter = "udp port 53"
	default:
		bpfFilter = filterInput
		fmt.Printf("Применяю пользовательский фильтр: '%s'\n", bpfFilter)
	}
	return bpfFilter
}

// Вспомогательная функция для получения IP-адресов
func getIPs(device pcap.Interface) string {
	if len(device.Addresses) == 0 {
		return "нет IP"
	}
	for _, addr := range device.Addresses {
		if addr.IP != nil {
			return addr.IP.String()
		}
	}
	return "нет IP"
}

func printFrame(packet gopacket.Packet, count int) {
	data := packet.Data()
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
		// Проверяем на VLAN (802.1Q): теги для виртуальных LAN.
		if vlan := packet.Layer(layers.LayerTypeDot1Q); vlan != nil {
			v := vlan.(*layers.Dot1Q)
			writeLine(fmt.Sprintf(" ↳ VLAN: ID %d (Приоритет %d)",
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

func hexDump(data []byte) {
	const bytesPerLine = 16
	for i := 0; i < len(data); i += bytesPerLine {
		line := data[i:min(i+bytesPerLine, len(data))]
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
		for len(hex) < 49 {
			hex += " "
		}
		prefix := " "
		if i == 0 {
			prefix = "Dst"
		} else if i == 6 {
			prefix = "Src"
		} else if i == 12 {
			prefix = "Type"
		}
		offsetStr := fmt.Sprintf("0x%02X", i)
		writeLine(fmt.Sprintf("%s %s %s %s", offsetStr, prefix, hex, ascii))
	}
}

// Вспомогательная функция для записи в консоль и файл
func writeLine(s string) {
	fmt.Println(s)
	outputFile.WriteString(s + "\n")
	if err := outputFile.Sync(); err != nil {
		// Ошибка записи: %v
		// Игнорируем ошибки записи для простоты
	}
}
