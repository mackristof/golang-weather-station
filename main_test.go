// main_test
package main

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

const fileName string = "./test-data.bin"

//func TestGettingDataFromWeatherStation(t *testing.T) {

//	actual := GetConnectionFromWeatherStation()
//	if assert.Equal(t, "/dev/ttyUSB0", actual) {
//		connection := ConnectStation()
//		assert.True(t, WakeUpStation(connection))
//		time.Sleep(time.Second * 2)
//		assert.Contains(t, CallTestSequence(connection), "TEST")
//		oneLoop := GetCurrentData(connection, 1)
//		//StoreCurrentDataToFile(fileName, oneLoop)
//		//file, err := os.Open(fileName)
//		//check(err)
//		//buffer := make([]byte, 99)
//		//nbRead, err := file.Read(buffer)
//		//check(err)
//		assert.Len(t, oneLoop, 99)
//		assert.Equal(t, []byte("LOO"), oneLoop[0:3])
//		//assert.True(t, nbRead == 99)
//		//assert.Equal(t, oneLoop, buffer)
//	}
//}

func readFromFile(t *testing.T, dataChannel chan<- *WeatherData, wg *sync.WaitGroup) {
	file, err := os.Open(fileName)
	check(err)
	buffer := make([]byte, 99)
	nbRead, err := file.Read(buffer)
	check(err)
	assert.True(t, nbRead == 99)
	weatherData := new(WeatherData)
	DecodeData(buffer, weatherData)
	log.Println("try to send msg to channel")
	dataChannel <- weatherData
	log.Println("send msg to channel")
	wg.Done()
}

func TestSendWeatherData(t *testing.T) {

	mqttClient := StartMqttConnection()
	var wg sync.WaitGroup
	var dataChannel chan *WeatherData = make(chan *WeatherData)
	wg.Add(2)
	go readFromMqttChannel(&wg)
	time.Sleep(3 * time.Second)
	go readFromFile(t, dataChannel, &wg)
	go PostCurrentData(dataChannel, mqttClient)

	wg.Wait()
	mqttClient.Disconnect(250)
}

func onMessageReceived(client *MQTT.MqttClient, message MQTT.Message) {
	log.Printf("Received message on topic: %s\n", message.Topic())
	log.Printf("Message: %s\n", message.Payload())
}

func readFromMqttChannel(wg *sync.WaitGroup) {
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
	check(err)
	for {
		time.Sleep(2 * time.Second)
		log.Println("sleeping for 2 sec")
	}
	c.Disconnect(250)
	wg.Done()
}
