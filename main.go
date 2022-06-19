package main

import (
	"fmt"
	"log"
	"strings"
	"time"
	"strconv"

	"go.bug.st/serial"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var port serial.Port
var err error

var (
	bedTemp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "f127_3d_printer_bed_temp",
		Help: "Bed temperature",
	})
	bedTempTarget = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "f127_3d_printer_bed_temp_target",
		Help: "Bed temperature target",
	})
	nozzleTemp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "f127_3d_printer_nozzle_temp",
		Help: "Nozzle temperature",
	})
	nozzleTempTarget = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "f127_3d_printer_nozzle_temp_target",
		Help: "Nozzle temperature target",
	})
)

func getFirmware(){
	buff := make([]byte, 1)
	firmwareLine := []byte{}
	_, err := port.Write([]byte("M115\n\r"))
	if err != nil {
		log.Fatal(err)
	}

	for {

		_, err := port.Read(buff)
		if err != nil {
			log.Fatal(err)
		}

		firmwareLine = append(firmwareLine, buff...)

		if strings.Contains(string(buff), "\n"){
			firmwareLineStr := string(firmwareLine[:])
			fmt.Print(firmwareLineStr)
			firmwareLine = nil

			if strings.Contains(firmwareLineStr,"ok\n"){
				break
			}
		}
	}
}

func readTemperature() {

	var nozzle_temp_whole int
	var nozzle_temp_minor_1 int

	var nozzle_temp_target_whole int
	var nozzle_temp_target_minor_1 int

	var bed_temp_whole int
	var bed_temp_minor_1 int

	var bed_temp_target_whole int
	var bed_temp_target_minor_1 int

	var dunno_1 string
	var dunno_2 string

	buff := make([]byte, 1)
	temperatureLine := []byte{}
	_, err := port.Write([]byte("M105\n\r"))
	if err != nil {
		log.Fatal(err)
	}
	
	
	for {
		_, err := port.Read(buff)
		if err != nil {
			log.Fatal(err)
		}
		
		temperatureLine = append(temperatureLine, buff...)

		if strings.Contains(string(buff), "\n"){
			temperatureLineStr := string(temperatureLine[:])
			fmt.Print(temperatureLineStr)
			temperatureLine = nil

			_, err := fmt.Sscanf(temperatureLineStr, 
				"ok T:%d.%d /%d.%d B:%d.%d /%d.%d @:%s B@:%s", 
				&nozzle_temp_whole, &nozzle_temp_minor_1, 
				&nozzle_temp_target_whole, &nozzle_temp_target_minor_1,
				&bed_temp_whole, &bed_temp_minor_1, 
				&bed_temp_target_whole, &bed_temp_target_minor_1,
				&dunno_1, &dunno_2)
				if err != nil {
					// panic(err)
					fmt.Println(temperatureLineStr)
				} else {
					nozzle_temp, _ := strconv.ParseFloat(fmt.Sprintf("%d.%d%d", nozzle_temp_whole, nozzle_temp_minor_1),64)
					nozzle_temp_target, _ := strconv.ParseFloat(fmt.Sprintf("%d.%d%d", nozzle_temp_target_whole, nozzle_temp_target_minor_1),64)
					bed_temp, _ := strconv.ParseFloat(fmt.Sprintf("%d.%d%d", bed_temp_whole, bed_temp_minor_1),64)
					bed_temp_target, _ := strconv.ParseFloat(fmt.Sprintf("%d.%d%d", bed_temp_target_whole, bed_temp_target_minor_1),64)
	
					fmt.Printf("Nozzle temp is %f | Target set to: %f\n",nozzle_temp, nozzle_temp_target)
					fmt.Printf("Bed temp is %f | Target set to: %f\n",bed_temp, bed_temp_target)
	
					nozzleTemp.Set(nozzle_temp)
					nozzleTempTarget.Set(nozzle_temp_target)
					bedTemp.Set(bed_temp)
					bedTempTarget.Set(bed_temp_target)
				}

			if strings.Contains(temperatureLineStr,"\n"){
				break
			}
		}
	}
	
}

func readPrinter() {
	for {
		buff := make([]byte, 100)
		for {
			time.Sleep(5 * time.Second)

			n, err := port.Read(buff)
			if err != nil {
				log.Fatal(err)
			}
			if n == 0 {
				fmt.Println("\nEOF")
				break
			}

			fmt.Printf("%s", string(buff[:n]))

			if strings.Contains(string(buff[:n]), "\n"){
				break
			}
		}
	}
}

func readPosition() {
	fmt.Println("position>")
	_, err := port.Write([]byte("M114\n\r"))
	if err != nil {
		log.Fatal(err)
	}
	
	for {
		buff := make([]byte, 1024)
		for {
			
	
			time.Sleep(5 * time.Second)
	
			n, err := port.Read(buff)
			if err != nil {
				log.Fatal(err)
			}
			if n == 0 {
				fmt.Println("\nEOF")
				break
			}
	
			fmt.Printf("%s", string(buff[:n]))
	
			if strings.Contains(string(buff[:n]), "\n"){
				fmt.Println("breaking")
				break
			}
		}
	}
}

func main(){

	mode := &serial.Mode{
		BaudRate: 115200,
		Parity: serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	port, err = serial.Open("/dev/ttyUSB0", mode)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Getting Firmware")
	getFirmware()
	
	go func() {
		for{
			fmt.Println("Getting Temperature")
			readTemperature()
			fmt.Println("Getting Position")
			readPosition()	
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112",nil)
	// readPrinter()
}