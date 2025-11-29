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

// selectBPFFilter: Выбор BPF-фильтра через меню.
// Принимает reader, возвращает string фильтра.
func selectBPFFilter(reader *bufio.Reader) string {
	fmt.Println("\nВыбери тип фильтра (BPF):")
	fmt.Println("[0] Всё (без фильтра)")
	fmt.Println("---------------------------------------------------------")
	fmt.Println("--- L2: Канальный уровень (Ethernet, ARP, Обнаружение) ---")
	fmt.Println("---------------------------------------------------------")
	fmt.Println("[1] Только ARP (сопоставление IP↔MAC)")
	fmt.Println("[2] Только ARP-запросы (Who has IP?)")
	fmt.Println("[3] Только LLDP (обнаружение соседей)")
	fmt.Println("[4] Только STP/RSTP (Spanning Tree, предотвращение петель)")
	fmt.Println("[5] Ethernet broadcast (широковещательные фреймы)")
	fmt.Println("[6] VLAN-tagged frames (помеченные VLAN)")
	fmt.Println("[7] CDP (Cisco Discovery Protocol, обнаружение)")
	fmt.Println("[8] Wake-on-LAN (WoL, магические пакеты)")
	fmt.Println("---------------------------------------------------------")
	fmt.Println("--- L3: Сетевой уровень (IP, ICMP, Маршрутизация) ---")
	fmt.Println("---------------------------------------------------------")
	fmt.Println("[9] ICMP (Ping-запросы и ответы)")
	fmt.Println("[10] OSPF (Протокол маршрутизации)")
	fmt.Println("[11] ВЕСЬ трафик на конкретный IP-адрес (Нужен ввод)")
	fmt.Println("[12] Весь НЕ-локальный трафик (Не 192.168.x.x, 10.x.x.x)")
	fmt.Println("---------------------------------------------------------")
	fmt.Println("--- L4: Транспортный уровень (TCP/UDP, Порты) ---")
	fmt.Println("---------------------------------------------------------")
	fmt.Println("[13] Весь TCP и UDP трафик")
	fmt.Println("[14] DNS-запросы (UDP порт 53)")
	fmt.Println("[15] HTTPS (TCP порт 443)")
	fmt.Println("[16] HTTP (TCP порт 80)")
	fmt.Println("[17] SSH (TCP порт 22)")
	fmt.Println("[18] RDP (Remote Desktop, TCP порт 3389)")
	fmt.Println("[19] SMB (File Share, TCP порт 445)")
	fmt.Println("[20] SMTP/POP3/IMAP (Email-трафик)")
	fmt.Println("[21] Syslog (UDP порт 514)")

	fmt.Print("\nИли введи свой фильтр (Пример: 'host 8.8.8.8 and not port 53'): ")

	filterInput, _ := reader.ReadString('\n')
	filterInput = strings.TrimSpace(filterInput)

	var bpfFilter string
	switch filterInput {
	case "0", "":
		bpfFilter = ""
	case "1":
		bpfFilter = "arp"
		fmt.Println("Применяю фильтр: 'arp'")
	case "2":
		bpfFilter = "arp and arp[7] == 1"
		fmt.Println("Применяю фильтр: 'arp and arp[7] == 1'")
	case "3":
		bpfFilter = "ether proto 0x88cc"
		fmt.Println("Применяю фильтр: 'ether proto 0x88cc'")
	case "4":
		bpfFilter = "stp or rstp"
		fmt.Println("Применяю фильтр: 'stp or rstp'")
	case "5":
		bpfFilter = "ether broadcast"
		fmt.Println("Применяю фильтр: 'ether broadcast'")
	case "6":
		bpfFilter = "vlan"
		fmt.Println("Применяю фильтр: 'vlan'")
	case "7":
		bpfFilter = "ether[12:2] == 0x2000"
		fmt.Println("Применяю фильтр: 'ether[12:2] == 0x2000'")
	case "8":
		bpfFilter = "ether[0:6] == 0xffffffff and udp port 9"
		fmt.Println("Применяю фильтр: 'ether[0:6] == 0xffffffff and udp port 9'")
	case "9":
		bpfFilter = "icmp or icmp6"
		fmt.Println("Применяю фильтр: 'icmp or icmp6'")
	case "10":
		bpfFilter = "proto ospf"
		fmt.Println("Применяю фильтр: 'proto ospf'")
	case "11":
		fmt.Print("Введите IP-адрес для фильтрации (например, 8.8.8.8): ")
		ipInput, _ := reader.ReadString('\n')
		ipInput = strings.TrimSpace(ipInput)
		if ipInput != "" {
			bpfFilter = fmt.Sprintf("host %s", ipInput)
			fmt.Printf("Применяю фильтр: 'host %s'\n", ipInput)
		} else {
			fmt.Println("IP-адрес не введен. Фильтр отменен.")
			bpfFilter = ""
		}
	case "12":
		bpfFilter = "not (net 192.168.0.0/16 or net 10.0.0.0/8 or net 172.16.0.0/12)"
		fmt.Println("Применяю фильтр: 'not (net 192.168.0.0/16 or net 10.0.0.0/8 or net 172.16.0.0/12)'")

	case "13":
		bpfFilter = "tcp or udp"
		fmt.Println("Применяю фильтр: 'tcp or udp'")
	case "14":
		bpfFilter = "udp port 53"
		fmt.Println("Применяю фильтр: 'udp port 53'")
	case "15":
		bpfFilter = "tcp port 443"
		fmt.Println("Применяю фильтр: 'tcp port 443'")
	case "16":
		bpfFilter = "tcp port 80"
		fmt.Println("Применяю фильтр: 'tcp port 80'")
	case "17":
		bpfFilter = "tcp port 22"
		fmt.Println("Применяю фильтр: 'tcp port 22'")
	case "18":
		bpfFilter = "tcp port 3389"
		fmt.Println("Применяю фильтр: 'tcp port 3389'")
	case "19":
		bpfFilter = "tcp port 445"
		fmt.Println("Применяю фильтр: 'tcp port 445'")
	case "20":
		bpfFilter = "tcp port 25 or tcp port 110 or tcp port 995 or tcp port 143 or tcp port 993"
		fmt.Println("Применяю фильтр: 'tcp port 25 or tcp port 110 or tcp port 995 or tcp port 143 or tcp port 993'")
	case "21":
		bpfFilter = "udp port 514"
		fmt.Println("Применяю фильтр: 'udp port 514'")
	default:
		bpfFilter = filterInput
		// Проверка фильтра будет выполнена в main
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
