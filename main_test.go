// main_test
package main

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const fileName string = "./test-data.bin"

func TestGettingDataFromWeatherStation(t *testing.T) {

	actual := GetConnectionFromWeatherStation()
	if assert.Equal(t, "/dev/ttyUSB0", actual) {
		connection := ConnectStation()
		assert.True(t, WakeUpStation(connection))
		time.Sleep(time.Second * 2)
		assert.Contains(t, CallTestSequence(connection), "TEST")
		oneLoop := GetCurrentData(connection, 1)
		//StoreCurrentDataToFile(fileName, oneLoop)
		//file, err := os.Open(fileName)
		//check(err)
		//buffer := make([]byte, 99)
		//nbRead, err := file.Read(buffer)
		//check(err)
		assert.Len(t, oneLoop, 99)
		assert.Equal(t, []byte("LOO"), oneLoop[0:3])
		//assert.True(t, nbRead == 99)
		//assert.Equal(t, oneLoop, buffer)
	}
}

func TestDecodeData(t *testing.T) {
	file, err := os.Open(fileName)
	check(err)
	buffer := make([]byte, 99)
	nbRead, err := file.Read(buffer)
	check(err)
	assert.True(t, nbRead == 99)
	log.Printf("test data: %q\n", buffer)
	result := DecodeData(buffer)
	assert.Empty(t, result)
}
