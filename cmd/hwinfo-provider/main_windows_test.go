package main

import (
	"encoding/binary"
	"math"
	"testing"

	"diagnostic-studio/internal/model"
)

func TestReadHWiNFOMemory(t *testing.T) {
	buf := make([]byte, 2048)
	const sensorOffset = 64
	const readingOffset = sensorOffset + 264

	binary.LittleEndian.PutUint32(buf[0:4], hwisSignature)
	binary.LittleEndian.PutUint32(buf[4:8], 1)
	binary.LittleEndian.PutUint32(buf[20:24], sensorOffset)
	binary.LittleEndian.PutUint32(buf[24:28], 264)
	binary.LittleEndian.PutUint32(buf[28:32], 1)
	binary.LittleEndian.PutUint32(buf[32:36], readingOffset)
	binary.LittleEndian.PutUint32(buf[36:40], 316)
	binary.LittleEndian.PutUint32(buf[40:44], 2)

	copy(buf[sensorOffset+8:sensorOffset+136], []byte("CPU [#0]"))

	temp := buf[readingOffset : readingOffset+316]
	binary.LittleEndian.PutUint32(temp[0:4], sensorTypeTemp)
	copy(temp[12:140], []byte("CPU Package"))
	copy(temp[268:284], []byte("C"))
	binary.LittleEndian.PutUint64(temp[284:292], math.Float64bits(87.25))

	power := buf[readingOffset+316 : readingOffset+632]
	binary.LittleEndian.PutUint32(power[0:4], sensorTypePower)
	copy(power[12:140], []byte("CPU Package Power"))
	copy(power[268:284], []byte("W"))
	binary.LittleEndian.PutUint64(power[284:292], math.Float64bits(24.5))

	h, err := readHeader(buf)
	if err != nil {
		t.Fatalf("readHeader: %v", err)
	}
	sensors, err := readSensors(buf, h)
	if err != nil {
		t.Fatalf("readSensors: %v", err)
	}
	readings, err := readReadings(buf, h, sensors)
	if err != nil {
		t.Fatalf("readReadings: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("readings=%d, want 2: %#v", len(readings), readings)
	}
	if readings[0].Kind != "temperature" || readings[0].Value != 87.25 {
		t.Fatalf("unexpected temperature reading: %#v", readings[0])
	}
	if readings[1].Kind != "power" || readings[1].Unit != "W" {
		t.Fatalf("unexpected power reading: %#v", readings[1])
	}
}

func TestReadHeaderRejectsBadSignature(t *testing.T) {
	buf := make([]byte, 128)
	binary.LittleEndian.PutUint32(buf[0:4], 0x44414544)
	if _, err := readHeader(buf); err == nil {
		t.Fatal("expected bad signature to fail")
	}
}

func TestSummarizeReadingsKeepsKeyMetrics(t *testing.T) {
	readings := summarizeReadings([]model.SensorReading{
		{Kind: "voltage", Name: "CPU [#0] / Core VID 0", Unit: "V", Value: 0.7},
		{Kind: "voltage", Name: "CPU [#0] / Core VID 1", Unit: "V", Value: 0.8},
		{Kind: "temperature", Name: "CPU [#0] / Core Temperatures", Unit: "°C", Value: 50},
		{Kind: "temperature", Name: "CPU [#0] / CPU Package", Unit: "°C", Value: 67},
		{Kind: "power", Name: "CPU [#0] / CPU Package Power", Unit: "W", Value: 38},
	})
	if len(readings) != 4 {
		t.Fatalf("readings=%d, want 4: %#v", len(readings), readings)
	}
	if readings[0].Name != "CPU Package Temperature" || readings[0].Value != 67 {
		t.Fatalf("unexpected first reading: %#v", readings[0])
	}
	if readings[2].Name != "CPU Package Power" {
		t.Fatalf("unexpected power reading order/name: %#v", readings)
	}
}
