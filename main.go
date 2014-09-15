// main project main.go
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tarm/goserial"
)

const ACK byte = '\x06'

func main() {
	connection := ConnectStation()
	WakeUpStation(connection)
	time.Sleep(time.Second * 2)
	oneLoop := GetCurrentData(connection, 1)
	fmt.Printf("data =%q\n", oneLoop)
}

//check if error apperring
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// scan USB tty from /dev folder
func GetConnectionFromWeatherStation() string {
	contents, _ := ioutil.ReadDir("/dev")

	// Look for what is mostly likely the DAVIS device
	for _, f := range contents {
		if strings.Contains(f.Name(), "tty.usbserial") ||
			strings.Contains(f.Name(), "ttyUSB") {
			return "/dev/" + f.Name()
		}
	}

	// Have not been able to find a USB device that 'looks'
	// like an Davis.
	return ""
}

// create connection to Davis Weather Station
func ConnectStation() io.ReadWriteCloser {
	connectionParameter := &serial.Config{Name: GetConnectionFromWeatherStation(), Baud: 19200}
	connection, err := serial.OpenPort(connectionParameter)
	check(err)
	return connection
}

// wake up davis Station
func WakeUpStation(connection io.ReadWriteCloser) bool {
	nb, err := connection.Write([]byte("\n"))
	check(err)

	buf := make([]byte, 10)
	nb, err = connection.Read(buf)
	check(err)
	log.Printf("%d bytes: %s\n", nb, string(buf))
	if nb == 2 {
		return true
	} else {
		return WakeUpStation(connection)
	}
}

// call test sequence to Davis Weather Station
// it must return "TEST"
func CallTestSequence(connection io.ReadWriteCloser) string {
	nb, err := connection.Write([]byte("TEST\n"))
	check(err)

	buf := make([]byte, 100)
	nb, err = connection.Read(buf)
	check(err)
	log.Printf("%d bytes: %s\n", nb, string(buf))
	return string(buf)
}

//get current Data Weather from connection, loopNumber : number of iteration
func GetCurrentData(connection io.ReadWriteCloser, loopNumber int16) []byte {
	nb, err := connection.Write([]byte("LOOP " + string(loopNumber) + "\n"))
	check(err)

	buf := make([]byte, 100)
	nb, err = connection.Read(buf)
	check(err)
	log.Printf("%d bytes: %q\n", nb, buf)
	if buf[0] == ACK {
		return buf[1:100]
	} else {
		return nil
	}
}

func StoreCurrentDataToFile(filePath string, data []byte) {

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		check(err)
		writeToFile(file, data)
	} else {
		file, err := os.Open(filePath)
		check(err)
		writeToFile(file, data)
	}
}

func writeToFile(file *os.File, data []byte) {
	defer file.Close()
	_, err := file.Write(data)
	check(err)
}
