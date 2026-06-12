# HWiNFO Sensor Provider

This optional helper reads HWiNFO Shared Memory and prints Diagnostic Studio's
normalized sensor JSON to stdout. It is read-only and does not install drivers.

Build locally:

```powershell
go build -o providers\sensor-provider.exe ./cmd/hwinfo-provider
```

Usage:

1. Start HWiNFO.
2. Open Sensors.
3. Enable Shared Memory in HWiNFO settings.
4. Run a Diagnostic Studio scan.

The main app discovers the helper via `PC_DIAG_SENSOR_PROVIDER` first, then
`providers\sensor-provider.exe` or `sensors\sensor-provider.exe` next to the
app/current working directory.
