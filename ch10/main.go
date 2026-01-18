package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []URL    `xml:"url"`
}

type URL struct {
	Loc string `xml:"loc"`
}

func main() {
	fmt.Println("Выбери режим проверки:")
	fmt.Println("  1  → Ввести ссылки вручную")
	fmt.Println("  2  → Читать из файла sitemap.xml (в той же папке)")
	fmt.Println("  3  → Загрузить sitemap автоматически с сайта")
	fmt.Print("\nТвой выбор (1/2/3): ")

	var mode int
	fmt.Scanln(&mode)

	var urls []string
	var err error

	switch mode {
	case 1:
		urls, err = readUrlsManually()
	case 2:
		urls, err = readSitemapFromFile("sitemap.xml")
	case 3:
		urls, err = fetchSitemapFromWeb("https://encantashop.fi/sitemap.xml")
	default:
		fmt.Println("Неверный выбор. Выход.")
		return
	}

	fmt.Println("Текущая папка:", os.Getenv("PWD"))
	files, _ := os.ReadDir(".")
	fmt.Println("Файлы в текущей папке:")
	for _, f := range files {
		fmt.Println("  -", f.Name())
	}
	if _, err := os.Stat("sitemap.xml"); os.IsNotExist(err) {
		fmt.Println("!!! ФАЙЛ sitemap.xml НЕ НАЙДЕН в текущей папке !!!")
	}

	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return
	}

	if len(urls) == 0 {
		fmt.Println("Не найдено ни одной ссылки для проверки.")
		return
	}

	fmt.Printf("\nНайдено %d ссылок. Начинаем проверку...\n\n", len(urls))

	client := &http.Client{Timeout: 12 * time.Second}

	for i, url := range urls {
		if i > 0 {
			time.Sleep(3200 * time.Millisecond) // ≈ 3.2 сек
		}

		fmt.Printf("%4d) %-65s → ", i+1, url)

		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("ERROR   %v\n", err)
			continue
		}

		code := resp.StatusCode
		resp.Body.Close()

		if code >= 200 && code < 400 {
			fmt.Printf("OK   (%d)\n", code)
		} else {
			fmt.Printf("BAD  (%d)\n", code)
		}
	}

	fmt.Println("\nГотово. Проверено:", len(urls), "страниц.")
}

func readUrlsManually() ([]string, error) {
	fmt.Println("\nВводи ссылки (по одной на строку).")
	fmt.Println("Пустая строка или Ctrl+C → закончить ввод\n")

	var urls []string
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("→ ")
		scanner.Scan()
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "http") {
			urls = append(urls, line)
		}
	}

	return urls, scanner.Err()
}

func readSitemapFromFile(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл %s: %w\n(положи sitemap.xml в ту же папку, где запускаешь программу)", filename, err)
	}
	defer f.Close()

	return parseSitemap(f)
}

func fetchSitemapFromWeb(url string) ([]string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("сервер вернул %s", resp.Status)
	}

	return parseSitemap(resp.Body)
}

func parseSitemap(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var s URLSet
	if err := xml.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(s.URLs))
	for _, u := range s.URLs {
		if u.Loc != "" {
			urls = append(urls, u.Loc)
		}
	}

	return urls, nil
}
