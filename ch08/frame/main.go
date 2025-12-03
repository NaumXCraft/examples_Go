// ═════════════════════════════════════════════════════════════════════════════
//                             СНАЙПЕР ПАКЕТОВ НА GO
//                       Захват и анализ Ethernet-кадров в реальном времени
// ═════════════════════════════════════════════════════════════════════════════
//
// Это консольная программа на Go, которая:
//   • Захватывает сетевой трафик с выбранного интерфейса (Npcap/WinPcap)
//   • Применяет BPF-фильтры (ARP, DNS, HTTPS, IP и т.д.) для фокусировки
//   • Выводит каждый кадр по уровням OSI: L2 (Ethernet), L3 (IP), L4 (TCP/UDP), L5 (Payload/DNS)
//   • Генерирует hex-дамп "как на проводе" (в стиле Wireshark: с префиксами Dst/Src/Type и ASCII)
//   • Сохраняет всё в frames.txt для анализа в блокноте/Notepad++
//   • Останавливается после 100 пакетов (для демо; можно убрать лимит)
//
//
// Версия: 1.1 (улучшенный вывод: четкие уровни, полный hex с ASCII, без ошибок)
// Требования: go1.21+, Npcap (Windows) или libpcap (Linux/macOS)
// Зависимости: go get github.com/google/gopacket
//
// Запуск:
//   go mod init packet-sniffer
//   go get github.com/google/gopacket
//   go run main.go
// ═════════════════════════════════════════════════════════════════════════════

package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// --- Глобальные переменные и Константы ---
// outputFile остаётся глобальной для упрощения записи логов из любой функции.
var outputFile *os.File

// Константы для выбора BPF-фильтра: делают код в selectBPFFilter более читабельным.
const (
	FilterNone       = "0"
	FilterARP        = "1"
	FilterARPRequest = "2"
	FilterLLDP       = "3"
	FilterSTP        = "4"
	FilterBroadcast  = "5"
	FilterVLAN       = "6"
	FilterCDP        = "7"
	FilterWoL        = "8"
	FilterICMP       = "9"
	FilterOSPF       = "10"
	FilterHost       = "11"
	FilterNotLocal   = "12"
	FilterTCPUDP     = "13"
	FilterDNS        = "14"
	FilterHTTPS      = "15"
	FilterHTTP       = "16"
	FilterSSH        = "17"
	FilterRDP        = "18"
	FilterSMB        = "19"
	FilterEmail      = "20"
	FilterSyslog     = "21"
)

// --- Основная Логика (main) ---

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
			bpfFilter = "" // Сбрасываем фильтр, если он не сработал
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

// --- Вспомогательные Функции ---

// readInput: Вспомогательная функция для чтения и очистки ввода.
func readInput(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// initOutputFile: Инициализация файла лога и вывод заголовка.
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
func findDevices() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// selectDevice: Интерактивный выбор сетевого интерфейса.
func selectDevice(reader *bufio.Reader, devices []pcap.Interface) pcap.Interface {
	fmt.Println("\nДоступные интерфейсы:")
	for i, device := range devices {
		fmt.Printf("[%d] %s - %s (IP: %s)\n", i, device.Name, device.Description, getIPs(device))
	}
	fmt.Print("Выбери номер интерфейса (Enter для авто-выбора с IP): ")

	input := readInput(reader) // Используем новую вспомогательную функцию

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
func openInterface(device pcap.Interface) (*pcap.Handle, error) {
	// 65536 - максимальная длина снимка (SnapLen), true - promiscuous mode (режим "все вижу")
	return pcap.OpenLive(device.Name, 65536, true, pcap.BlockForever)
}

// selectBPFFilter: Выбор BPF-фильтра через меню.
func selectBPFFilter(reader *bufio.Reader) string {
	fmt.Println("\nВыбери тип фильтра (BPF):")
	fmt.Printf("[%s] Всё (без фильтра)\n", FilterNone)
	fmt.Println("---------------------------------------------------------")
	fmt.Println("--- L2: Канальный уровень (Ethernet, ARP, Обнаружение) --")
	fmt.Println("---------------------------------------------------------")
	fmt.Printf("[%s] Только ARP (сопоставление IP↔MAC)\n", FilterARP)
	fmt.Printf("[%s] Только ARP-запросы (Who has IP?)\n", FilterARPRequest)
	fmt.Printf("[%s] Только LLDP (обнаружение соседей)\n", FilterLLDP)
	fmt.Printf("[%s] Только STP/RSTP (Spanning Tree, предотвращение петель)\n", FilterSTP)
	fmt.Printf("[%s] Ethernet broadcast (широковещательные фреймы)\n", FilterBroadcast)
	fmt.Printf("[%s] VLAN-tagged frames (помеченные VLAN)\n", FilterVLAN)
	fmt.Printf("[%s] CDP (Cisco Discovery Protocol, обнаружение)\n", FilterCDP)
	fmt.Printf("[%s] Wake-on-LAN (WoL, магические пакеты)\n", FilterWoL)
	fmt.Println("---------------------------------------------------------")
	fmt.Println("--- L3: Сетевой уровень (IP, ICMP, Маршрутизация) -------")
	fmt.Println("---------------------------------------------------------")
	fmt.Printf("[%s] ICMP (Ping-запросы и ответы)\n", FilterICMP)
	fmt.Printf("[%s] OSPF (Протокол маршрутизации)\n", FilterOSPF)
	fmt.Printf("[%s] ВЕСЬ трафик на конкретный IP-адрес (Нужен ввод)\n", FilterHost)
	fmt.Printf("[%s] Весь НЕ-локальный трафик (Не 192.168.x.x, 10.x.x.x)\n", FilterNotLocal)
	fmt.Println("---------------------------------------------------------")
	fmt.Println("--- L4: Транспортный уровень (TCP/UDP, Порты) -----------")
	fmt.Println("---------------------------------------------------------")
	fmt.Printf("[%s] Весь TCP и UDP трафик\n", FilterTCPUDP)
	fmt.Printf("[%s] DNS-запросы (UDP порт 53)\n", FilterDNS)
	fmt.Printf("[%s] HTTPS (TCP порт 443)\n", FilterHTTPS)
	fmt.Printf("[%s] HTTP (TCP порт 80)\n", FilterHTTP)
	fmt.Printf("[%s] SSH (TCP порт 22)\n", FilterSSH)
	fmt.Printf("[%s] RDP (Remote Desktop, TCP порт 3389)\n", FilterRDP)
	fmt.Printf("[%s] SMB (File Share, TCP порт 445)\n", FilterSMB)
	fmt.Printf("[%s] SMTP/POP3/IMAP (Email-трафик)\n", FilterEmail)
	fmt.Printf("[%s] Syslog (UDP порт 514)\n", FilterSyslog)

	fmt.Print("\nИли введи свой фильтр (Пример: 'host 8.8.8.8 and not port 53'): ")

	filterInput := readInput(reader) // Используем новую вспомогательную функцию

	var bpfFilter string
	switch filterInput {
	case FilterNone, "":
		bpfFilter = ""
	case FilterARP:
		bpfFilter = "arp"
	case FilterARPRequest:
		bpfFilter = "arp and arp[7] == 1"
	case FilterLLDP:
		bpfFilter = "ether proto 0x88cc"
	case FilterSTP:
		bpfFilter = "ether proto 0x0000 or ether proto 0x0035"
	case FilterBroadcast:
		bpfFilter = "ether broadcast"
	case FilterVLAN:
		bpfFilter = "vlan"
	case FilterCDP:
		bpfFilter = "ether[12:2] == 0x2000"
	case FilterWoL:
		bpfFilter = "ether[0:6] == 0xffffffff and udp port 9" // Простой WoL-фильтр
	case FilterICMP:
		bpfFilter = "icmp or icmp6"
	case FilterOSPF:
		bpfFilter = "proto ospf"
	case FilterHost:
		fmt.Print("Введите IP-адрес для фильтрации (например, 8.8.8.8): ")
		ipInput := readInput(reader) // Используем вспомогательную функцию
		if ipInput != "" {
			bpfFilter = fmt.Sprintf("host %s", ipInput)
		} else {
			fmt.Println("IP-адрес не введен. Фильтр отменен.")
			bpfFilter = ""
		}
	case FilterNotLocal:
		bpfFilter = "not (net 192.168.0.0/16 or net 10.0.0.0/8 or net 172.16.0.0/12)"
	case FilterTCPUDP:
		bpfFilter = "tcp or udp"
	case FilterDNS:
		bpfFilter = "udp port 53"
	case FilterHTTPS:
		bpfFilter = "tcp port 443"
	case FilterHTTP:
		bpfFilter = "tcp port 80"
	case FilterSSH:
		bpfFilter = "tcp port 22"
	case FilterRDP:
		bpfFilter = "tcp port 3389"
	case FilterSMB:
		bpfFilter = "tcp port 445"
	case FilterEmail:
		bpfFilter = "tcp port 25 or tcp port 110 or tcp port 995 or tcp port 143 or tcp port 993"
	case FilterSyslog:
		bpfFilter = "udp port 514"
	default:
		bpfFilter = filterInput
	}

	if bpfFilter != "" && filterInput != FilterNone && filterInput != "" {
		fmt.Printf("Применяю фильтр: '%s'\n", bpfFilter)
	}

	return bpfFilter
}

// getIPs: Вспомогательная функция для получения IP-адресов устройства.
func getIPs(device pcap.Interface) string {
	if len(device.Addresses) == 0 {
		return "нет IP"
	}
	for _, addr := range device.Addresses {
		if addr.IP != nil && addr.IP.To4() != nil { // Предпочитаем IPv4
			return addr.IP.String()
		}
	}
	for _, addr := range device.Addresses { // Если нет IPv4, берем IPv6
		if addr.IP != nil {
			return addr.IP.String()
		}
	}
	return "нет IP"
}

// printFrame: Выводит анализ пакета по уровням OSI (L2-L5) + hex-дамп.
func printFrame(packet gopacket.Packet, count int) {
	data := packet.Data()
	writeLine(strings.Repeat("=", 80))
	writeLine(fmt.Sprintf("#%d | Время: %s | Общая Длина: %d байт",
		count, time.Now().Format("15:04:05.000"), len(data)))

	// L2: Канальный уровень (Ethernet, MAC-адреса, EtherType, VLAN)
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		eth := ethLayer.(*layers.Ethernet)
		writeLine(fmt.Sprintf(
			"L2 (Ethernet): %s → %s (EtherType: %s 0x%04x)",
			eth.SrcMAC, eth.DstMAC, eth.EthernetType, uint16(eth.EthernetType),
		))
		// Проверяем на VLAN (802.1Q)
		if vlan := packet.Layer(layers.LayerTypeDot1Q); vlan != nil {
			v := vlan.(*layers.Dot1Q)
			writeLine(fmt.Sprintf(" ↳ VLAN: ID %d (Приоритет %d)",
				v.VLANIdentifier, v.Priority))
		}
	} else {
		writeLine("L2: Не Ethernet (возможно, другой тип фрейма)")
	}

	// L3: Сетевой уровень (IP-адреса, маршрутизация, TTL/HopLimit)
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
	} else {
		// ARP, RARP и другие протоколы L2/L3 без IP-заголовка
		if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
			arp := arpLayer.(*layers.ARP)
			op := "REQUEST"
			if arp.Operation == layers.ARPReply {
				op = "REPLY"
			}
			writeLine(fmt.Sprintf("L3 (ARP): Операция: %s, %s (MAC: %s) → %s (MAC: %s)",
				op, arp.SourceProtAddress, arp.SourceHwAddress, arp.DstProtAddress, arp.DstHwAddress))
		} else {
			writeLine("L3: Нет сетевого уровня (локальный трафик?)")
		}
	}

	// L4: Транспортный уровень (TCP/UDP, порты, Seq/Ack, флаги)
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
	} else {
		writeLine("L4: Нет транспортного уровня (не TCP/UDP)")
	}

	// L5: Прикладной уровень (DNS, HTTP, TLS, прочий payload)
	if appLayer := packet.ApplicationLayer(); appLayer != nil {
		if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
			dns := dnsLayer.(*layers.DNS)
			if len(dns.Questions) > 0 {
				q := dns.Questions[0]
				writeLine(fmt.Sprintf(
					"L5 (DNS): Запрос: %s (Тип %s)",
					q.Name, q.Type,
				))
			} else if len(dns.Answers) > 0 {
				writeLine(fmt.Sprintf("L5 (DNS): Ответ (ID: %d)", dns.ID))
			}
		}
		// Payload прикладного уровня (например HTTP, TLS, RAW данные)
		payload := appLayer.Payload()
		if len(payload) > 0 {
			writeLine(fmt.Sprintf("L5 (Payload): Длина: %d байт", len(payload)))
		}
	} else {
		writeLine("L5: Нет прикладного уровня (сырые данные или зашифровано)")
	}

	writeLine("----------------------------------------------------")
	writeLine("Полный кадр в hex :")
	hexDump(data)
	writeLine("")
}

// getTCPFlags: Формирует строку с флагами TCP (FIN/SYN/RST и т.д.).
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

// hexDump: Выводит hex-дамп байтов в стиле Wireshark (с offset, префиксами, hex и ASCII).
func hexDump(data []byte) {
	const bytesPerLine = 16
	// Полезный контекст для новичка:
	// 0x00-0x05: MAC-адрес получателя (Dst)
	// 0x06-0x0B: MAC-адрес отправителя (Src)
	// 0x0C-0x0D: EtherType (тип протокола вышележащего уровня, например IPv4: 0x0800)

	for i := 0; i < len(data); i += bytesPerLine {
		end := int(math.Min(float64(i+bytesPerLine), float64(len(data))))
		line := data[i:end]
		hex := ""
		ascii := ""
		for j, b := range line {
			hex += fmt.Sprintf("%02X ", b)
			if b >= 32 && b <= 126 {
				ascii += string(b)
			} else {
				ascii += "."
			}
			if j == 7 { // Двойной пробел между 8-м и 9-м байтом для лучшей читаемости
				hex += " "
			}
		}
		// Выравнивание hex
		for len(hex) < 49 {
			hex += "  "
		}

		// Префиксы для первых строк (как в Wireshark)
		prefix := ""
		if i == 0 {
			prefix = "Dst "
		} else if i == 6 {
			prefix = "Src "
		} else if i == 12 {
			prefix = "Type"
		}
		offsetStr := fmt.Sprintf("0x%04X", i) // Используем 4 цифры для смещения для единообразия
		writeLine(fmt.Sprintf("%s %s%s %s", offsetStr, prefix, hex, ascii))
	}
}

// writeLine: Вспомогательная функция для записи в консоль и файл (с синхронизацией).
func writeLine(s string) {
	fmt.Println(s)
	// В проде нужно добавить более надежную обработку ошибок, но для простоты игнорируем.
	if _, err := outputFile.WriteString(s + "\n"); err != nil {
		// Log error if writing fails
	}
	if err := outputFile.Sync(); err != nil {
		// Log error if sync fails
	}
}
