// Package enable communation within Davis Vantage weather station & can emit data on mqtt broker
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"

	"github.com/mackristof/golang-weather-station/weatherStation"
)

func main() {
	connection, err := weatherStation.Connect()
	check(err)
	weatherStation.WakeUp(connection)
	mqttClient := StartMqttConnection()
	for {
		var wg sync.WaitGroup
		var dataChannel chan *weatherStation.WeatherData = make(chan *weatherStation.WeatherData)
		wg.Add(1)
		go weatherStation.GetCurrentData(connection, 30, dataChannel, &wg)
		go PostCurrentData(dataChannel, mqttClient)
		wg.Wait()
	}

	log.Println("END")
}

//check if error apperring
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// post WeatherData structure on JSON to MQTT topic
func PostCurrentData(dataChannel <-chan *weatherStation.WeatherData, mqttClient *MQTT.MqttClient) {
	log.Println("start posting data")
	for {
		currentWeather := <-dataChannel
		fmt.Printf("#########################getting data =%+v\n", currentWeather)
		jsonWeather, err := json.Marshal(currentWeather)
		fmt.Printf("#########################sended data =%s\n", jsonWeather)
		check(err)
		msg := MQTT.NewMessage(jsonWeather)
		mqttClient.PublishMessage("/mackristof/weather-mtp/davis1", msg)
		//log.Printf("posted msg : %+v\n", msg)
	}
}

func StartMqttConnection() *MQTT.MqttClient {
	opts := MQTT.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientId("GolangWeatherStation")
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
