package collector

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"diagnostic-studio/internal/model"
)

// powercfg output is localized, so parsing only relies on locale-independent
// tokens: the scheme GUID, the parenthesized display name, and hex setting
// indexes. For a single-setting query the hex values appear in a fixed order:
// min, max, increment, then current AC, then current DC — the last two are
// what we want, on every locale.

var (
	reGUID = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	reName = regexp.MustCompile(`\(([^()]+)\)\s*$`)
	reHex  = regexp.MustCompile(`:\s*0x([0-9a-fA-F]+)\s*$`)
)

const powerSaverGUID = "a1841308-3541-4fab-bc81-f71556f20b4a"

func collectPower(notes *[]string) model.PowerStateInfo {
	p := model.PowerStateInfo{
		ACMaxProcessorState: -1,
		DCMaxProcessorState: -1,
		ACBoostMode:         -1,
		DCBoostMode:         -1,
		BatteryWearPercent:  -1,
		ThermalZoneMaxC:     -1,
	}

	if out, err := runPS("powercfg /getactivescheme"); err == nil {
		p.ActiveSchemeGUID = strings.ToLower(reGUID.FindString(out))
		if m := reName.FindStringSubmatch(strings.TrimSpace(out)); len(m) == 2 {
			p.ActiveSchemeName = m[1]
		}
	} else {
		*notes = append(*notes, "powercfg active scheme: "+err.Error())
	}

	if ac, dc, err := queryPowerSetting("SUB_PROCESSOR", "PROCTHROTTLEMAX"); err == nil {
		p.ACMaxProcessorState, p.DCMaxProcessorState = ac, dc
	} else {
		*notes = append(*notes, "powercfg max processor state: "+err.Error())
	}
	if ac, dc, err := queryPowerSetting("SUB_PROCESSOR", "PERFBOOSTMODE"); err == nil {
		p.ACBoostMode, p.DCBoostMode = ac, dc
	} else {
		*notes = append(*notes, "powercfg boost mode: "+err.Error())
	}

	p.OnAC = readACLineStatus(notes)
	collectBattery(&p, notes)
	collectThermal(&p, notes)
	return p
}

// queryPowerSetting returns the current AC and DC indexes of one setting in
// the active scheme. Hidden settings (e.g. PERFBOOSTMODE) are absent from /q
// output, so /qh is tried as a fallback.
func queryPowerSetting(subgroup, setting string) (ac, dc int, err error) {
	for _, flag := range []string{"/q", "/qh"} {
		out, runErr := runPS(fmt.Sprintf("powercfg %s SCHEME_CURRENT %s %s", flag, subgroup, setting))
		if runErr != nil {
			err = runErr
			continue
		}
		var values []int
		for _, line := range strings.Split(out, "\n") {
			if m := reHex.FindStringSubmatch(strings.TrimRight(line, "\r ")); len(m) == 2 {
				v, perr := strconv.ParseInt(m[1], 16, 32)
				if perr == nil {
					values = append(values, int(v))
				}
			}
		}
		// Current AC and DC indexes are the only 0x-prefixed values; enum
		// definitions use plain decimal ("Possible Setting Index: 002").
		if len(values) >= 2 {
			return values[len(values)-2], values[len(values)-1], nil
		}
		err = fmt.Errorf("unexpected powercfg output for %s (%d hex values)", setting, len(values))
	}
	return -1, -1, err
}

// IsPowerSaverScheme reports whether a scheme GUID is the built-in power
// saver plan. Exported for the analyzer.
func IsPowerSaverScheme(guid string) bool {
	return strings.EqualFold(guid, powerSaverGUID)
}

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

func readACLineStatus(notes *[]string) bool {
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemPowerStatus")
	var st systemPowerStatus
	r, _, callErr := proc.Call(uintptr(unsafe.Pointer(&st)))
	if r == 0 {
		*notes = append(*notes, "GetSystemPowerStatus: "+callErr.Error())
		return true // assume AC rather than fabricate a battery signal
	}
	return st.ACLineStatus == 1
}

// BatteryStaticData needs elevation; values are coerced to numbers so the
// JSON shape stays stable whether or not the queries succeed.
const batteryScript = `
$full = 0; $design = 0
try { $full = [int64](Get-CimInstance -Namespace root/wmi -ClassName BatteryFullChargedCapacity -ErrorAction Stop | Select-Object -First 1 -ExpandProperty FullChargedCapacity) } catch {}
try { $design = [int64](Get-CimInstance -Namespace root/wmi -ClassName BatteryStaticData -ErrorAction Stop | Select-Object -First 1 -ExpandProperty DesignedCapacity) } catch {}
@{full=$full; design=$design} | ConvertTo-Json
`

func collectBattery(p *model.PowerStateInfo, notes *[]string) {
	var b struct {
		Full   int `json:"full"`
		Design int `json:"design"`
	}
	if err := runPSJSON(batteryScript, &b); err != nil {
		*notes = append(*notes, "battery capacity: "+err.Error())
		return
	}
	if b.Full <= 0 || b.Design <= 0 {
		return // desktop or battery WMI not exposed
	}
	p.BatteryPresent = true
	p.BatteryFullMWh = b.Full
	p.BatteryDesignedMWh = b.Design
	p.BatteryWearPercent = (1 - float64(b.Full)/float64(b.Design)) * 100
	if p.BatteryWearPercent < 0 {
		p.BatteryWearPercent = 0
	}
}

const thermalScript = `
$z = Get-CimInstance -Namespace root/wmi -ClassName MSAcpi_ThermalZoneTemperature -ErrorAction SilentlyContinue | Select-Object -ExpandProperty CurrentTemperature
if ($null -eq $z) { '[]' } else { ConvertTo-Json -InputObject @($z) }
`

func collectThermal(p *model.PowerStateInfo, notes *[]string) {
	var temps []float64
	if err := runPSJSON(thermalScript, &temps); err != nil {
		*notes = append(*notes, "thermal zone: "+err.Error())
		return
	}
	maxC := -1.0
	for _, t := range temps {
		c := t/10 - 273.15
		if c > maxC && c > 0 && c < 130 {
			maxC = c
		}
	}
	p.ThermalZoneMaxC = maxC
}
