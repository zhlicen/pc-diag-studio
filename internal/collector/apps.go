package collector

import (
	"strings"

	"diagnostic-studio/internal/model"
)

const appsScript = `
$paths = @(
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\Software\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*'
)
$apps = Get-ItemProperty $paths -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -and -not $_.SystemComponent } |
    Select-Object DisplayName, DisplayVersion, Publisher |
    Sort-Object DisplayName -Unique
ConvertTo-Json -InputObject @($apps) -Depth 2
`

// Duplicate-utility categories. Edge/Defender are excluded: preinstalled
// components shouldn't count toward "the user installed N of these".
var utilityCategories = map[string][]string{
	"browser":   {"chrome", "firefox", "opera", "brave", "vivaldi", "360安全浏览器", "360极速浏览器", "qq浏览器", "搜狗高速浏览器", "uc浏览器", "猎豹浏览器", "世界之窗"},
	"archive":   {"winrar", "7-zip", "bandizip", "winzip", "haozip", "好压", "360压缩", "快压", "2345好压", "peazip"},
	"assistant": {"360安全卫士", "腾讯电脑管家", "电脑管家", "鲁大师", "驱动精灵", "驱动人生", "360软件管家", "2345安全卫士", "金山毒霸", "cleanmymac", "ccleaner", "advanced systemcare"},
}

func categorize(name string) string {
	lower := strings.ToLower(name)
	for cat, keywords := range utilityCategories {
		for _, k := range keywords {
			if strings.Contains(lower, k) {
				return cat
			}
		}
	}
	return ""
}

func collectInstalledApps(notes *[]string) []model.InstalledApp {
	var raw []struct {
		DisplayName    string `json:"DisplayName"`
		DisplayVersion string `json:"DisplayVersion"`
		Publisher      string `json:"Publisher"`
	}
	if err := runPSJSON(appsScript, &raw); err != nil {
		*notes = append(*notes, "installed apps: "+err.Error())
		return nil
	}
	apps := make([]model.InstalledApp, 0, len(raw))
	for _, a := range raw {
		apps = append(apps, model.InstalledApp{
			Name:      a.DisplayName,
			Version:   a.DisplayVersion,
			Publisher: a.Publisher,
			Category:  categorize(a.DisplayName),
		})
	}
	return apps
}
