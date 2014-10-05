// main_test
package main_test

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mackristof/golang-weather-station/main"
	"github.com/mackristof/golang-weather-station/weatherStation"
)

const fileName string = "./test-data.bin"

func TestGettingDataFromWeatherStation(t *testing.T) {

	actual := weatherStation.GetUSBPath()
	if assert.Equal(t, "/dev/ttyUSB0", actual) {
		connection, err := weatherStation.Connect()
		assert.Nil(t, err)
		assert.True(t, weatherStation.WakeUp(connection))
		time.Sleep(time.Second * 2)
		assert.Contains(t, weatherStation.CallTestSequence(connection), "TEST")
		var wg sync.WaitGroup
		var dataChannel chan *weatherStation.WeatherData = make(chan *weatherStation.WeatherData)
		wg.Add(1)
		go weatherStation.GetCurrentData(connection, 1, dataChannel, &wg)
		wg.Wait()
		oneLoop := <-dataChannel
		assert.Len(t, oneLoop, 40)
		assert.Equal(t, 12, oneLoop.AvgWindSpeed)
		//assert.True(t, nbRead == 99)
		//assert.Equal(t, oneLoop, buffer)
	}
}

func readFromFile(t *testing.T, dataChannel chan<- *weatherStation.WeatherData, wg *sync.WaitGroup) {
	file, err := os.Open(fileName)
	assert.Nil(t, err)
	buffer := make([]byte, 99)
	nbRead, err := file.Read(buffer)
	assert.Nil(t, err)
	assert.True(t, nbRead == 99)
	weatherData := new(weatherStation.WeatherData)
	weatherStation.DecodeData(buffer, weatherData)
	log.Println("try to send msg to channel")
	dataChannel <- weatherData
	log.Println("send msg to channel")
	wg.Done()
}

func TestSendWeatherData(t *testing.T) {

	mqttClient := main.StartMqttConnection()
	var wg sync.WaitGroup
	var dataChannel chan *weatherStation.WeatherData = make(chan *weatherStation.WeatherData)
	wg.Add(2)
	go readFromMqttChannel(t, &wg)
	time.Sleep(3 * time.Second)
	go readFromFile(t, dataChannel, &wg)
	go main.PostCurrentData(dataChannel, mqttClient)

	wg.Wait()
	mqttClient.Disconnect(250)
}

func onMessageReceived(client *MQTT.MqttClient, message MQTT.Message) {
	log.Printf("Received message on topic: %s\n", message.Topic())
	log.Printf("Message: %s\n", message.Payload())
}

func readFromMqttChannel(t *testing.T, wg *sync.WaitGroup) {
	log.Println("try to connect to MQTT for getting data")
	opts := MQTT.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientId("myReader")

	c := MQTT.NewClient(opts)
	_, err := c.Start()
	log.Println("client connected")
	if err != nil {
		panic(err)
	}

	filter, _ := MQTT.NewTopicFilter("/mackristof/weather-mtp/davis1", 1)
	_, err = c.StartSubscription(onMessageReceived, filter)
	assert.Nil(t, err)
	for {
		time.Sleep(2 * time.Second)
		log.Println("sleeping for 2 sec")
	}
	c.Disconnect(250)
	wg.Done()
}
