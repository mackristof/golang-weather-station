// main project main.go
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/tarm/goserial"
)

const ACK byte = '\x06'

// data struture of current weather
type WeatherData struct {
	barTrend        int8
	packetType      int8
	nextRecord      int16
	barometer       float64
	insideTemp      float64
	insideHumidity  int8
	outsideTemp     float64
	windSpeed       float64
	avgWindSpeed    float64
	windDirection   int16
	outsideHumidity int8
	rainRate        float64
}

func main() {
	connection := ConnectStation()
	WakeUpStation(connection)
	mqttClient := StartMqttConnection()
	var wg sync.WaitGroup
	var dataChannel chan *WeatherData = make(chan *WeatherData)
	wg.Add(1)
	go GetCurrentData(connection, 30, dataChannel, &wg)
	go PostCurrentData(dataChannel, mqttClient)
	wg.Wait()
	log.Println("END")
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
		time.Sleep(time.Second * 2)
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
			DecodeData(buf, weatherData)
			dataChannel <- weatherData

			log.Printf("loop: %d \n", i)
			time.Sleep(time.Second * 2)
			i--
		}
	} else {
		log.Fatal("can't get data from weather Station")
	}
	wg.Done()
	return nil
}

// decode []byte from weather station to WeatherData stucture
func DecodeData(buffer []byte, weatherData *WeatherData) {

	//get barTrend
	buf := bytes.NewReader(buffer[3:4])
	err := binary.Read(buf, binary.LittleEndian, &weatherData.barTrend)
	check(err)

	//get packetType
	buf = bytes.NewReader(buffer[4:5])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.packetType)
	check(err)

	//get nextRecord
	buf = bytes.NewReader(buffer[5:7])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.nextRecord)
	check(err)
	log.Printf("nextRecord: %d", weatherData.nextRecord)

	//get barometer
	buf = bytes.NewReader(buffer[7:9])
	var barometer int16
	err = binary.Read(buf, binary.LittleEndian, &barometer)
	check(err)
	weatherData.barometer = ((float64(barometer) / 1000) * 33.8638)
	log.Printf("Barometer: %f", weatherData.barometer)

	// get inside temp in Celsius
	buf = bytes.NewReader(buffer[9:11])
	var tempInF int16
	err = binary.Read(buf, binary.LittleEndian, &tempInF)
	check(err)
	weatherData.insideTemp = float64(((tempInF / 10) - 32)) * (5 / 9.)
	log.Printf("Inside Temperature: %f", weatherData.insideTemp)

	// get inside Humidity
	buf = bytes.NewReader(buffer[11:12])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.insideHumidity)
	check(err)

	// get outside Temp in Celsius
	buf = bytes.NewReader(buffer[12:14])
	err = binary.Read(buf, binary.LittleEndian, &tempInF)
	check(err)
	weatherData.outsideTemp = float64(((tempInF / 10) - 32)) * (5 / 9.)
	log.Printf("Outside Temperature: %f", weatherData.outsideTemp)

	//get wind speed in km/h
	var windSpeed uint8
	buf = bytes.NewReader(buffer[14:15])
	err = binary.Read(buf, binary.LittleEndian, &windSpeed)
	check(err)
	weatherData.windSpeed = (float64(windSpeed) * 1.609344)
	log.Printf("windSpeed: %d", weatherData.windSpeed)

	//get 10 min avg  wind speed in km/h
	buf = bytes.NewReader(buffer[15:16])
	err = binary.Read(buf, binary.LittleEndian, &windSpeed)
	check(err)
	weatherData.avgWindSpeed = (float64(windSpeed) * 1.609344)
	log.Printf("avgWindSpeed: %d", weatherData.windSpeed)

	//get wind Direction
	buf = bytes.NewReader(buffer[16:18])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.windDirection)
	check(err)

	// get outside Humidity
	buf = bytes.NewReader(buffer[33:34])
	err = binary.Read(buf, binary.LittleEndian, &weatherData.outsideHumidity)
	check(err)

	//get rain rate in mm/h
	var rainRate int16
	buf = bytes.NewReader(buffer[41:43])
	err = binary.Read(buf, binary.LittleEndian, &rainRate)
	check(err)
	weatherData.rainRate = float64(rainRate) * 0.2

	//log.Println(weatherData)
}

// post WeatherData structure on JSON to MQTT topic
func PostCurrentData(dataChannel <-chan *WeatherData, mqttClient *MQTT.MqttClient) {
	for {
		currentWeather := <-dataChannel
		fmt.Printf("#########################sended data =%q\n", currentWeather.barometer)
		jsonWeather, err := json.Marshal(currentWeather)
		check(err)
		msg := MQTT.NewMessage(jsonWeather)
		mqttClient.PublishMessage("/test/weather", msg)
	}
}

func StartMqttConnection() *MQTT.MqttClient {
	opts := MQTT.NewClientOptions().AddBroker("tcp://test.mosquitto.org:1883").SetClientId("GolangWeatherStation")
	//opts.SetDefaultPublishHandler(f)

	c := MQTT.NewClient(opts)
	_, err := c.Start()
	check(err)
	return c
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
