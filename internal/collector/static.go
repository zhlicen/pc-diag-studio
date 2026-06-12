package collector

import (
	"strings"

	"diagnostic-studio/internal/model"
)

const staticScript = `
$cs = Get-CimInstance Win32_ComputerSystem | Select-Object -First 1 Manufacturer,Model,Name,TotalPhysicalMemory
$os = Get-CimInstance Win32_OperatingSystem | Select-Object -First 1 Caption,Version,BuildNumber
$cpu = Get-CimInstance Win32_Processor | Select-Object -First 1 Name,MaxClockSpeed,NumberOfCores,NumberOfLogicalProcessors
$disks = @(Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | Select-Object DeviceID,VolumeName,Size,FreeSpace)
@{cs=$cs; os=$os; cpu=$cpu; disks=$disks; sysDrive=$env:SystemDrive} | ConvertTo-Json -Depth 4
`

type staticPayload struct {
	CS struct {
		Manufacturer        string `json:"Manufacturer"`
		Model               string `json:"Model"`
		Name                string `json:"Name"`
		TotalPhysicalMemory uint64 `json:"TotalPhysicalMemory"`
	} `json:"cs"`
	OS struct {
		Caption     string `json:"Caption"`
		Version     string `json:"Version"`
		BuildNumber string `json:"BuildNumber"`
	} `json:"os"`
	CPU struct {
		Name                      string `json:"Name"`
		MaxClockSpeed             int    `json:"MaxClockSpeed"`
		NumberOfCores             int    `json:"NumberOfCores"`
		NumberOfLogicalProcessors int    `json:"NumberOfLogicalProcessors"`
	} `json:"cpu"`
	Disks []struct {
		DeviceID   string  `json:"DeviceID"`
		VolumeName string  `json:"VolumeName"`
		Size       float64 `json:"Size"`
		FreeSpace  float64 `json:"FreeSpace"`
	} `json:"disks"`
	SysDrive string `json:"sysDrive"`
}

func collectStatic() (model.ComputerInfo, model.CPUInfo, model.MemoryInfo, []model.DiskInfo, error) {
	var p staticPayload
	if err := runPSJSON(staticScript, &p); err != nil {
		return model.ComputerInfo{}, model.CPUInfo{}, model.MemoryInfo{}, nil, err
	}

	computer := model.ComputerInfo{
		ComputerName: p.CS.Name,
		Manufacturer: p.CS.Manufacturer,
		Model:        p.CS.Model,
		OSName:       p.OS.Caption,
		OSVersion:    p.OS.Version,
		OSBuild:      p.OS.BuildNumber,
	}
	cpu := model.CPUInfo{
		Name:              strings.TrimSpace(p.CPU.Name),
		BaseClockMHz:      p.CPU.MaxClockSpeed,
		Cores:             p.CPU.NumberOfCores,
		LogicalProcessors: p.CPU.NumberOfLogicalProcessors,
	}
	mem := model.MemoryInfo{TotalMB: int(p.CS.TotalPhysicalMemory / (1024 * 1024))}

	var disks []model.DiskInfo
	for _, d := range p.Disks {
		if d.Size <= 0 {
			continue
		}
		disks = append(disks, model.DiskInfo{
			Drive:       d.DeviceID,
			Label:       d.VolumeName,
			TotalGB:     d.Size / (1024 * 1024 * 1024),
			FreeGB:      d.FreeSpace / (1024 * 1024 * 1024),
			FreePercent: d.FreeSpace / d.Size * 100,
			IsSystem:    strings.EqualFold(d.DeviceID, p.SysDrive),
		})
	}
	return computer, cpu, mem, disks, nil
}
