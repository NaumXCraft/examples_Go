package main

import (
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

/*
================================================================================
Go + Wireshark BPF: структурированное руководство
================================================================================

1. Ключевой момент
------------------
handle.SetBPFFilter(filter) — основной метод для установки BPF (Capture Filter).

Как работает:
- Go передаёт текстовую строку фильтра в библиотеку pcap.
- pcap компилирует её в байт-код (BPF-программу).
- Драйвер сетевой карты (Npcap/libpcap) применяет фильтр на уровне ядра.
- В цикл захвата пакетов попадут только прошедшие фильтр пакеты:

for packet := range packetSource.Packets() { ... }

2. Где ставить фильтр
---------------------
В начале кода задаём константу filter:

BPF-фильтр (можно менять)
const filter = "" // пустой = ловим все пакеты

Меняйте только filter, остальной код main() остаётся неизменным.

3. Примеры фильтров BPF
-----------------------
| Что ловим               | BPF-фильтр                  | Пояснение |
|-------------------------|-----------------------------|--------------------------------------------------|
| Весь веб-трафик         | port 80 or port 443         | HTTP и HTTPS пакеты |
| Отправитель MAC         | ether src 00:00:5e:00:53:aa | Пакеты с конкретного MAC |
| Только ARP-запросы      | arp and arp[7] = 1          | Канальный уровень (L2) |
| Трафик к сети 10.x.x.x  | dst net 10.0.0.0/8          | Весь трафик в сеть 10.0.0.0/8 |
| Всё кроме DNS           | not udp port 53             | Ловим всё, кроме DNS |

4. Советы
----------
- Для теста используйте пустой фильтр: const filter = "".
- На Windows запускайте от имени администратора.
- Проверяйте правильный интерфейс (IPv4 адаптера).
- Для локального трафика (127.0.0.1) используйте Npcap Loopback Adapter.
================================================================================
*/

// BPF-фильтр — пустой = ловим все пакеты
const filter = ""

// Рабочий интерфейс Npcap (IP address)
const localIPv4 = "192.168.1.100"

// Рабочий интерфейс Npcap (name interface)
// const interfaceName = `\Device\NPF_{E2AE0BA3-01EA-4D2A-83DA-C201E6A6CD36}`

// Таймаут захвата (секунды)
const captureTimeout = 15

func main() {
	// 1. Найти интерфейс по IPv4
	iface, err := findInterfaceByIPv4(localIPv4)
	if err != nil {
		fmt.Println("Ошибка поиска интерфейса:", err)
		return
	}

	// 2. Открыть интерфейс
	handle, err := openInterface(iface)
	if err != nil {
		fmt.Println("Ошибка открытия интерфейса:", err)
		return
	}
	defer handle.Close()

	// 3. Применить BPF-фильтр
	if err := applyBPF(handle, filter); err != nil {
		fmt.Println("Ошибка применения фильтра:", err)
		return
	}

	// 4. Запустить захват пакетов
	startCapture(handle, 10, captureTimeout)
}

// ----------------------- Функции -----------------------

// findInterfaceByIPv4 ищет интерфейс по IPv4-адресу
func findInterfaceByIPv4(ip string) (pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return pcap.Interface{}, err
	}

	for _, d := range devices {
		for _, a := range d.Addresses {
			if a.IP.String() == ip {
				fmt.Printf("Найден интерфейс: %s (%s)\n", d.Name, d.Description)
				return d, nil
			}
		}
	}

	return pcap.Interface{}, fmt.Errorf("интерфейс с IP %s не найден", ip)
}

// openInterface открывает выбранный интерфейс
func openInterface(iface pcap.Interface) (*pcap.Handle, error) {
	handle, err := pcap.OpenLive(iface.Name, 65536, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	fmt.Println("Интерфейс открыт:", iface.Name)
	return handle, nil
}

// applyBPF применяет BPF-фильтр к handle
func applyBPF(handle *pcap.Handle, filter string) error {
	if err := handle.SetBPFFilter(filter); err != nil {
		return err
	}
	fmt.Println("Фильтр BPF применён:", filter)
	return nil
}

// startCapture захватывает пакеты и выводит их читаемо
func startCapture(handle *pcap.Handle, maxPackets int, timeoutSec int) {
	fmt.Println("Захват пакетов запущен. Ctrl+C для остановки.")
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	counter := 0
	timeout := time.After(time.Duration(timeoutSec) * time.Second)

	for {
		select {
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}
			counter++
			printPacketReadable(counter, packet)

			if counter >= maxPackets {
				fmt.Printf("\nЗахвачено %d пакетов — остановка.\n", maxPackets)
				return
			}
		case <-timeout:
			fmt.Println("Таймаут завершён — пакеты не поступали.")
			return
		}
	}
}

// printPacketReadable красиво выводит TCP/UDP пакеты
func printPacketReadable(counter int, packet gopacket.Packet) {
	if tr := packet.TransportLayer(); tr != nil && packet.NetworkLayer() != nil {
		srcIP, dstIP := packet.NetworkLayer().NetworkFlow().Endpoints()
		switch t := tr.(type) {
		case *layers.TCP:
			fmt.Printf("\n#%d TCP %s:%d -> %s:%d Flags[SYN=%v ACK=%v FIN=%v PSH=%v RST=%v] Len=%d\n",
				counter,
				srcIP, t.SrcPort,
				dstIP, t.DstPort,
				t.SYN, t.ACK, t.FIN, t.PSH, t.RST,
				len(t.Payload))
		case *layers.UDP:
			fmt.Printf("\n#%d UDP %s:%d -> %s:%d Len=%d\n",
				counter,
				srcIP, t.SrcPort,
				dstIP, t.DstPort,
				len(t.Payload))
		default:
			fmt.Printf("\n#%d Transport: %v\n", counter, tr)
		}
	} else {
		fmt.Printf("\n#%d Нет транспортного слоя\n", counter)
	}

	if app := packet.ApplicationLayer(); app != nil {
		fmt.Printf("   Application payload: %d bytes\n", len(app.Payload()))
	}
}
