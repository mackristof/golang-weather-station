// Package enable communation within Davis Vantage weather station & can emit data on mqtt broker
package main

import (
	"encoding/json"
	//"log"
	"os"
	"sync"
	"time"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"

	log "github.com/cihub/seelog"
	"github.com/mackristof/golang-weather-station/weatherStation"
)

var mqttConnected = false
var choke chan [2]string

func main() {
	defer log.Flush()
	logConfig := `
<seelog>
	<outputs formatid="format1">
		<rollingfile type="size" filename="./log/roll.log" maxsize="100" maxrolls="5" />
	</outputs>
	<formats>
		<format id="format1" format="%Date/%Time [%LEV] %Msg%n"/>
	</formats>
</seelog>
`
	logger, err := log.LoggerFromConfigAsBytes([]byte(logConfig))
	check(err)
	log.ReplaceLogger(logger)
	weatherStation.UseLogger(logger)
	defer weatherStation.FlushLog()

	connection, err := weatherStation.Connect()
	check(err)
	weatherStation.WakeUp(connection)
	mqttClient, opts, err := StartMqttConnection()
	check(err)
	for {
		var wg sync.WaitGroup
		var dataChannel chan *weatherStation.WeatherData = make(chan *weatherStation.WeatherData)
		wg.Add(1)
		go weatherStation.GetCurrentData(connection, 30, dataChannel, &wg)
		go PostCurrentData(dataChannel, mqttClient, opts)
		wg.Wait()
		mqttClient.Disconnect(100)
		time.Sleep(time.Second * 1)
		var connected bool = false
		var i int = 1
		for connected == false {
			mqttClient, _, err = StartMqttConnection()
			if err == nil {
				connected = true
			} else {
				time.Sleep(time.Second * time.Duration(i))
				log.Warnf("wait for %d second to try to reconnect mqtt broker", i)
				i++
			}
		}

		//time.Sleep(time.Second * 5)
	}

	log.Info("END")
}

//check if error apperring
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// post WeatherData structure on JSON to MQTT topic
func PostCurrentData(dataChannel <-chan *weatherStation.WeatherData, mqttClient *MQTT.MqttClient, opts *MQTT.ClientOptions) {
	//log.Println("start posting data")
	for {
		currentWeather := <-dataChannel
		//fmt.Printf("#########################getting data =%+v\n", currentWeather)
		jsonWeather, err := json.Marshal(currentWeather)
		//fmt.Printf("#########################sended data =%s\n", jsonWeather)
		check(err)
		msg := MQTT.NewMessage(jsonWeather)
		reconnectMqttBroker(mqttClient, opts)
		if mqttConnected {
			mqttClient.PublishMessage("/mackristof/weather-mtp/davis1", msg)
		}
	}
}

func StartMqttConnection() (*MQTT.MqttClient, *MQTT.ClientOptions, error) {
	opts := MQTT.NewClientOptions().AddBroker("tcp://test.mosquitto.org:1883").SetClientId("GolangWeatherStation")
	opts.SetOnConnectionLost(onMqttConnLost)
	c := MQTT.NewClient(opts)
	_, err := c.Start()
	mqttConnected = true
	return c, opts, err
}

func reconnectMqttBroker(client *MQTT.MqttClient, opts *MQTT.ClientOptions) {
	if mqttConnected == false {
		//log.Println("Attempting reconnect . . .")
		// Always reconnect with cleansession false
		opts.SetCleanSession(false)
		_, err := client.Start()
		if err != nil {
			time.Sleep(1 * time.Second)
		} else {
			log.Info("Reconnected")
			mqttConnected = true
		}
	}
}

func onMqttConnLost(client *MQTT.MqttClient, err error) {
	//log.Println("Connection lost!")
	if mqttConnected == true {
		//log.Println("Connection lost!")
		mqttConnected = false
		choke <- [2]string{"DISCONNECTED", "DISCONNECTED"}
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
