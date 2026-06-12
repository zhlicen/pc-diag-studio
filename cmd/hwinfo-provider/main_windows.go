package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"

	"diagnostic-studio/internal/model"
)

const (
	mapName   = `Global\HWiNFO_SENS_SM2`
	mutexName = `Global\HWiNFO_SM2_MUTEX`

	hwisSignature = 0x53695748 // "HWiS" in little-endian memory
	maxMapBytes   = 16 << 20

	sensorTypeTemp  = 1
	sensorTypeVolt  = 2
	sensorTypeFan   = 3
	sensorTypePower = 5
)

type sharedHeader struct {
	Signature            uint32
	Version              uint32
	Revision             uint32
	PollTime             int64
	SensorSectionOffset  uint32
	SensorElementSize    uint32
	SensorElementCount   uint32
	ReadingSectionOffset uint32
	ReadingElementSize   uint32
	ReadingElementCount  uint32
}

type sensorElement struct {
	ID       uint32
	Inst     uint32
	NameOrig string
	NameUser string
}

func main() {
	snap, err := collect()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(snap); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func collect() (model.SensorSnapshot, error) {
	mappingName, err := syscall.UTF16PtrFromString(mapName)
	if err != nil {
		return model.SensorSnapshot{}, err
	}
	h, err := openFileMapping(windows.FILE_MAP_READ, false, mappingName)
	if err != nil {
		return model.SensorSnapshot{
			Provider: "hwinfo-shared-memory",
			Status:   "absent",
			Detail:   "advanced sensors: HWiNFO shared memory not available; enable Sensors and Shared Memory in HWiNFO",
		}, nil
	}
	defer windows.CloseHandle(h)

	mutex, releaseMutex := openMutex()
	if mutex != 0 {
		defer windows.CloseHandle(mutex)
	}
	if releaseMutex != nil {
		defer releaseMutex()
	}

	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ, 0, 0, 0)
	if err != nil {
		return model.SensorSnapshot{}, fmt.Errorf("map HWiNFO shared memory: %w", err)
	}
	defer windows.UnmapViewOfFile(addr)

	data := unsafe.Slice((*byte)(unsafe.Pointer(addr)), maxMapBytes)
	header, err := readHeader(data)
	if err != nil {
		return model.SensorSnapshot{}, err
	}
	sensors, err := readSensors(data, header)
	if err != nil {
		return model.SensorSnapshot{}, err
	}
	readings, err := readReadings(data, header, sensors)
	if err != nil {
		return model.SensorSnapshot{}, err
	}
	if len(readings) == 0 {
		return model.SensorSnapshot{}, errors.New("HWiNFO returned no supported CPU/fan readings")
	}
	readings = summarizeReadings(readings)

	return model.SensorSnapshot{
		Provider:   "hwinfo-shared-memory",
		Status:     "ok",
		CapturedAt: pollTime(header.PollTime),
		Readings:   readings,
	}, nil
}

func readHeader(data []byte) (sharedHeader, error) {
	if len(data) < 44 {
		return sharedHeader{}, errors.New("HWiNFO shared memory is too small")
	}
	h := sharedHeader{
		Signature:            binary.LittleEndian.Uint32(data[0:4]),
		Version:              binary.LittleEndian.Uint32(data[4:8]),
		Revision:             binary.LittleEndian.Uint32(data[8:12]),
		PollTime:             int64(binary.LittleEndian.Uint64(data[12:20])),
		SensorSectionOffset:  binary.LittleEndian.Uint32(data[20:24]),
		SensorElementSize:    binary.LittleEndian.Uint32(data[24:28]),
		SensorElementCount:   binary.LittleEndian.Uint32(data[28:32]),
		ReadingSectionOffset: binary.LittleEndian.Uint32(data[32:36]),
		ReadingElementSize:   binary.LittleEndian.Uint32(data[36:40]),
		ReadingElementCount:  binary.LittleEndian.Uint32(data[40:44]),
	}
	if h.Signature != hwisSignature {
		return sharedHeader{}, fmt.Errorf("unexpected HWiNFO signature 0x%08x", h.Signature)
	}
	if h.SensorElementSize < 264 || h.ReadingElementSize < 292 {
		return sharedHeader{}, errors.New("unsupported HWiNFO shared memory element size")
	}
	if !rangeOK(len(data), h.SensorSectionOffset, h.SensorElementSize, h.SensorElementCount) ||
		!rangeOK(len(data), h.ReadingSectionOffset, h.ReadingElementSize, h.ReadingElementCount) {
		return sharedHeader{}, errors.New("HWiNFO shared memory offsets are outside mapped range")
	}
	return h, nil
}

func readSensors(data []byte, h sharedHeader) (map[uint32]sensorElement, error) {
	out := make(map[uint32]sensorElement, h.SensorElementCount)
	for i := uint32(0); i < h.SensorElementCount; i++ {
		base := int(h.SensorSectionOffset + h.SensorElementSize*i)
		raw := data[base : base+int(h.SensorElementSize)]
		item := sensorElement{
			ID:       binary.LittleEndian.Uint32(raw[0:4]),
			Inst:     binary.LittleEndian.Uint32(raw[4:8]),
			NameOrig: cString(raw[8:136]),
			NameUser: cString(raw[136:264]),
		}
		out[i] = item
	}
	return out, nil
}

func readReadings(data []byte, h sharedHeader, sensors map[uint32]sensorElement) ([]model.SensorReading, error) {
	var out []model.SensorReading
	for i := uint32(0); i < h.ReadingElementCount; i++ {
		base := int(h.ReadingSectionOffset + h.ReadingElementSize*i)
		raw := data[base : base+int(h.ReadingElementSize)]
		kindCode := binary.LittleEndian.Uint32(raw[0:4])
		sensorIndex := binary.LittleEndian.Uint32(raw[4:8])
		label := firstNonEmpty(cString(raw[140:268]), cString(raw[12:140]))
		unit := cString(raw[268:284])
		value := math.Float64frombits(binary.LittleEndian.Uint64(raw[284:292]))
		if !validValue(value) {
			continue
		}

		kind, ok := normalizeReading(kindCode, unit)
		if !ok {
			continue
		}
		sensorName := ""
		if sensor, ok := sensors[sensorIndex]; ok {
			sensorName = firstNonEmpty(sensor.NameUser, sensor.NameOrig)
		}
		if !wantedReading(kind, label, sensorName) {
			continue
		}
		name := label
		if sensorName != "" {
			name = sensorName + " / " + label
		}
		out = append(out, model.SensorReading{
			Kind:   kind,
			Name:   name,
			Unit:   unit,
			Value:  value,
			Source: "hwinfo-shared-memory",
		})
	}
	return out, nil
}

func summarizeReadings(readings []model.SensorReading) []model.SensorReading {
	selected := make(map[string]model.SensorReading)
	for _, r := range readings {
		slot := readingSlot(r)
		if slot == "" {
			continue
		}
		prev, ok := selected[slot]
		if !ok || readingScore(r) > readingScore(prev) {
			selected[slot] = r
		}
	}
	order := []string{"cpu-package-temp", "cpu-core-temp", "cpu-package-power", "cpu-ia-power", "cpu-voltage", "fan"}
	out := make([]model.SensorReading, 0, len(order))
	for _, slot := range order {
		if r, ok := selected[slot]; ok {
			r.Name = displayName(slot, r.Name)
			out = append(out, r)
		}
	}
	if len(out) > 0 {
		return out
	}

	sort.SliceStable(readings, func(i, j int) bool {
		return readingScore(readings[i]) > readingScore(readings[j])
	})
	if len(readings) > 8 {
		return readings[:8]
	}
	return readings
}

func readingSlot(r model.SensorReading) string {
	hay := strings.ToLower(r.Name)
	switch r.Kind {
	case "temperature":
		switch {
		case strings.Contains(hay, "package") || strings.Contains(hay, "tctl") || strings.Contains(hay, "tdie"):
			return "cpu-package-temp"
		case strings.Contains(hay, "core max") || strings.Contains(hay, "core temperatures"):
			return "cpu-core-temp"
		case strings.Contains(hay, "core"):
			return "cpu-core-temp"
		}
	case "power":
		switch {
		case strings.Contains(hay, "package"):
			return "cpu-package-power"
		case strings.Contains(hay, "ia cores") || strings.Contains(hay, "cpu cores") || strings.Contains(hay, "core power"):
			return "cpu-ia-power"
		}
	case "voltage":
		if strings.Contains(hay, "vid") || strings.Contains(hay, "vcore") || strings.Contains(hay, "core") {
			return "cpu-voltage"
		}
	case "fan":
		return "fan"
	}
	return ""
}

func readingScore(r model.SensorReading) int {
	name := strings.ToLower(r.Name)
	score := 0
	if strings.Contains(name, "package") {
		score += 50
	}
	if strings.Contains(name, "core max") || strings.Contains(name, "maximum") {
		score += 30
	}
	if strings.Contains(name, "avg") || strings.Contains(name, "average") {
		score += 20
	}
	if strings.Contains(name, "vid") || strings.Contains(name, "vcore") {
		score += 20
	}
	if strings.Contains(name, "cpu [#0]") {
		score += 10
	}
	return score
}

func displayName(slot, fallback string) string {
	switch slot {
	case "cpu-package-temp":
		return "CPU Package Temperature"
	case "cpu-core-temp":
		return "CPU Core Temperature"
	case "cpu-package-power":
		return "CPU Package Power"
	case "cpu-ia-power":
		return "CPU Core Power"
	case "cpu-voltage":
		return "CPU Core Voltage"
	case "fan":
		return "Fan Speed"
	default:
		return fallback
	}
}

func normalizeReading(kindCode uint32, unit string) (string, bool) {
	switch kindCode {
	case sensorTypeTemp:
		return "temperature", true
	case sensorTypeVolt:
		return "voltage", true
	case sensorTypeFan:
		return "fan", true
	case sensorTypePower:
		return "power", true
	default:
		return "", false
	}
}

func wantedReading(kind, label, sensor string) bool {
	hay := strings.ToLower(sensor + " " + label)
	switch kind {
	case "temperature":
		return strings.Contains(hay, "cpu") || strings.Contains(hay, "package") ||
			strings.Contains(hay, "core") || strings.Contains(hay, "tctl") || strings.Contains(hay, "tdie")
	case "power":
		return strings.Contains(hay, "cpu") || strings.Contains(hay, "package") ||
			strings.Contains(hay, "core") || strings.Contains(hay, "ppt")
	case "voltage":
		return strings.Contains(hay, "cpu") || strings.Contains(hay, "core") ||
			strings.Contains(hay, "vid") || strings.Contains(hay, "vcore")
	case "fan":
		return true
	default:
		return false
	}
}

func cString(raw []byte) string {
	if i := strings.IndexByte(string(raw), 0); i >= 0 {
		raw = raw[:i]
	}
	s := strings.TrimSpace(string(raw))
	if !utf8.ValidString(s) {
		return ""
	}
	return s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func pollTime(t int64) string {
	if t <= 0 {
		return time.Now().Format(time.RFC3339)
	}
	return time.Unix(t, 0).Format(time.RFC3339)
}

func rangeOK(total int, offset, size, count uint32) bool {
	if size == 0 {
		return count == 0
	}
	end := uint64(offset) + uint64(size)*uint64(count)
	return end <= uint64(total)
}

func validValue(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
