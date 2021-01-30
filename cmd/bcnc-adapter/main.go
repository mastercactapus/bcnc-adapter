package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"go.bug.st/serial"
)

func sender(addr string, ch chan string) {
	var resp *http.Response
	var err error
	for cmd := range ch {
		if cmd == "STOP" {
			resp, err = http.Post(addr+"/send?cmd=stop", "", nil)
		} else {
			resp, err = http.Post(addr+"/send?gcode="+url.QueryEscape(cmd), "", nil)
		}
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Println("ERROR: non-200:", resp.Status)
		}
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	addr := flag.String("addr", "http://127.0.0.1:8080", "bCNC pendant server address.")
	port := flag.String("port", "", "Port to open")
	baud := flag.Int("baud", 115200, "Serial baudrate.")
	flag.Parse()

	sp, err := serial.Open(*port, &serial.Mode{BaudRate: *baud})
	if err != nil {
		log.Fatal(err)
	}
	defer sp.Close()

	sendCh := make(chan string, 100)
	go sender(*addr, sendCh)

	r := bufio.NewScanner(sp)
	for r.Scan() {
		parts := strings.Split(strings.TrimSpace(r.Text()), ":")
		switch parts[0] {
		case "STOP":
		case "STEP":
			var axis, mag, step int
			_, err = fmt.Sscanf("%d,%d,%d", parts[1], &axis, &mag, &step)
			if err != nil {
				log.Println("invalid data '%s': %v", parts[1], err)
				continue
			}
			var axisChr rune
			switch axis {
			case 1:
				axisChr = 'X'
			case 2:
				axisChr = 'Y'
			case 3:
				axisChr = 'Z'
			default:
				continue
			}

			select {
			case sendCh <- fmt.Sprintf("$J=G21G91F10000%c%0.4g\n", axisChr, float64(step)*float64(mag)/100):
			default:
			}
		}
	}

	log.Fatal(r.Err())
}
