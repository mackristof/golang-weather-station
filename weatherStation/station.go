package weatherStation

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tarm/goserial"
)

const ACK byte = '\x06'

// data struture of current weather
type WeatherData struct {
	BarTrend        int8
	PacketType      int8
	NextRecord      int16
	Barometer       float64
	InsideTemp      float64
	InsideHumidity  int8
	OutsideTemp     float64
	WindSpeed       float64
	AvgWindSpeed    float64
	WindDirection   int16
	OutsideHumidity int8
	RainRate        float64
}

// ConnectStation create connection to Davis Weather Station
func Connect() (io.ReadWriteCloser, error) {
	usbPorts := GetUSBPath()
	for _, usbPath := range usbPorts {
		connectionParameter := &serial.Config{Name: usbPath, Baud: 19200}
		connection, err := serial.OpenPort(connectionParameter)
		if err == nil {
			return connection, nil
		}
	}
	return nil, errors.New("cannot retrieve Davis Weather station on USB ports")
}

// getUSBPath scan USB tty from /dev folder
func GetUSBPath() []string {
	var usbPorts []string
	contents, _ := ioutil.ReadDir("/dev")
	// Look for what is mostly likely the DAVIS device
	for _, f := range contents {
		if strings.Contains(f.Name(), "tty.usbserial") ||
			strings.Contains(f.Name(), "ttyUSB") {
			usbPorts = append(usbPorts, "/dev/"+f.Name())
		}
	}

	// Have not been able to find a USB device that 'looks'
	// like an Davis.
	return usbPorts
}

// WakeUp davis Station with sending bites sequence
func WakeUp(connection io.ReadWriteCloser) bool {
	nb, err := connection.Write([]byte("\n"))
	check(err)

	buf := make([]byte, 10)
	nb, err = connection.Read(buf)
	check(err)
	log.Printf("%d bytes: %s\n", nb, string(buf))
	if nb == 2 {
		time.Sleep(time.Second * 2)
		return true
	} else {
		return WakeUp(connection)
	}
}

//GetCurrentData retreive data Weather from connection, loopNumber : number of iteration and send data to datachannel

func GetCurrentData(connection io.ReadWriteCloser, loopNumber int16, dataChannel chan<- *WeatherData, wg *sync.WaitGroup) error {
	_, err := connection.Write([]byte("LOOP " + string(loopNumber) + "\n"))
	check(err)
	i := loopNumber
	ackBuf := make([]byte, 1)
	_, err = connection.Read(ackBuf)
	check(err)
	log.Printf("bytes collected : %q\n", ackBuf)
	if ackBuf[0] == ACK {
		for i >= 1 {
			buf := make([]byte, 99)
			nb, err := connection.Read(buf)
			check(err)
			log.Printf("%d bytes collected : %q\n", nb, buf)

			weatherData := new(WeatherData)
			err = DecodeData(buf, weatherData)
			if err == nil {
				dataChannel <- weatherData
				log.Printf("loop: %d \n", i)
				time.Sleep(time.Second * 2)
				i--
			}

		}
	} else {
		log.Fatal("can't get data from weather Station")
	}
	wg.Done()
	return nil
}

// decodeData return []byte from weather station to WeatherData stucture
func DecodeData(buffer []byte, weatherData *WeatherData) error {
	//var header []byte
	//headerBuff := bytes.NewReader(buffer[0:2])
	//_, errHeader := headerBuff.Read(buffer[0:2])
	////errHeader := binary.Read(headerBuff, binary.LittleEndian, header)
	//check(errHeader)
	headerString := string(buffer[0:3])
	if headerString != "LOO" {
		log.Printf("bad header %v ", headerString)
		return errors.New("Bad header")
	}

	//get barTrend
	buf := bytes.NewReader(buffer[3:4])
	err := binary.Read(buf, binary.LittleEndian, &weatherData.BarTrend)
	check(err)

	//get packetType
	buf = bytes.NewReader(buffer[4:5])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.PacketType)
	check(err)

	//get nextRecord
	buf = bytes.NewReader(buffer[5:7])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.NextRecord)
	check(err)
	log.Printf("nextRecord: %d", weatherData.NextRecord)

	//get barometer
	buf = bytes.NewReader(buffer[7:9])
	var barometer int16
	err = binary.Read(buf, binary.LittleEndian, &barometer)
	check(err)
	weatherData.Barometer = ((float64(barometer) / 1000) * 33.8638)
	log.Printf("Barometer: %f", weatherData.Barometer)

	// get inside temp in Celsius
	buf = bytes.NewReader(buffer[9:11])
	var tempInF int16
	err = binary.Read(buf, binary.LittleEndian, &tempInF)
	check(err)
	weatherData.InsideTemp = float64(((tempInF / 10) - 32)) * (5 / 9.)
	log.Printf("Inside Temperature: %f", weatherData.InsideTemp)

	// get inside Humidity
	buf = bytes.NewReader(buffer[11:12])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.InsideHumidity)
	check(err)

	// get outside Temp in Celsius
	buf = bytes.NewReader(buffer[12:14])
	err = binary.Read(buf, binary.LittleEndian, &tempInF)
	check(err)
	weatherData.OutsideTemp = float64(((tempInF / 10) - 32)) * (5 / 9.)
	log.Printf("Outside Temperature: %f", weatherData.OutsideTemp)

	//get wind speed in km/h
	var windSpeed uint8
	buf = bytes.NewReader(buffer[14:15])
	err = binary.Read(buf, binary.LittleEndian, &windSpeed)
	check(err)
	weatherData.WindSpeed = (float64(windSpeed) * 1.609344)
	log.Printf("windSpeed: %d", weatherData.WindSpeed)

	//get 10 min avg  wind speed in km/h
	buf = bytes.NewReader(buffer[15:16])
	err = binary.Read(buf, binary.LittleEndian, &windSpeed)
	check(err)
	weatherData.AvgWindSpeed = (float64(windSpeed) * 1.609344)
	log.Printf("avgWindSpeed: %d", weatherData.AvgWindSpeed)

	//get wind Direction
	buf = bytes.NewReader(buffer[16:18])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.WindDirection)
	check(err)

	// get outside Humidity
	buf = bytes.NewReader(buffer[33:34])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.OutsideHumidity)
	check(err)

	//get rain rate in mm/h
	var rainRate int16
	buf = bytes.NewReader(buffer[41:43])
	err = binary.Read(buf, binary.LittleEndian, &rainRate)
	check(err)
	weatherData.RainRate = float64(rainRate) * 0.2
	return nil
	//log.Println(weatherData)
}

// CallTestSequence to Davis Weather Station
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

//check if error apperring
func check(e error) {
	if e != nil {
		panic(e)
	}
}
