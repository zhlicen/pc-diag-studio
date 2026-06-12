// Localized copy tables keyed by stable IDs (ruleId / causeId / evidenceId).
// The backend never sends display text — only IDs plus params. Adding a
// language means adding one entry to COPY.

export const COPY = {
  zh: {
    ui: {
      appName: 'Diagnostic Studio',
      subtitle: '诊断工作台',
      quickScan: '快速扫描 (15秒)',
      deepScan: '3 分钟深度扫描',
      scanning: '扫描中',
      scanHint: '扫描期间请像平时一样使用电脑,卡顿瞬间正是采样想抓住的',
      score: '体检评分',
      primary: '主要结论',
      attribution: '根因归因',
      attributionEmpty: '证据不足,无法给出可靠归因——建议在卡顿发生时运行 3 分钟深度扫描',
      findings: '诊断发现',
      evidence: '证据',
      samplingTitle: 'CPU 有效频率走势',
      admin: '管理员',
      notAdmin: '非管理员(诊断能力受限)',
      openLog: '打开日志',
      machine: '机器',
      noReport: '选择一种扫描模式开始诊断',
      collectorNotes: '采集说明',
      severity: { good: '良好', warning: '需要关注', critical: '严重' },
      findingSeverity: { info: '提示', warning: '警告', critical: '严重' },
      confidence: { high: '高置信', medium: '中置信', low: '低置信' },
      avgLoad: '平均负载',
      avgClock: '平均频率',
      minClock: '最低频率',
      memUsed: '内存占用',
      diskActive: '磁盘活动',
    },
    rules: {
      'rule.cpu-freq-constrained': {
        title: 'CPU 频率受限',
        desc: p => `采样期间 CPU 平均只跑到基准频率的 ${p.avgFreqRatioPercent}%,${p.lowFreqSamplePercent}% 的样本低于 55% 基准——卡顿更可能来自降频,而不是表面的资源占用。`,
      },
      'rule.power-saver-scheme': {
        title: '正在使用节能电源计划',
        desc: p => `当前电源计划「${p.schemeName}」会限制处理器响应速度。`,
      },
      'rule.memory-pressure': {
        title: '内存压力',
        desc: p => `内存平均占用 ${p.avgUsedPercent}%(峰值 ${p.maxUsedPercent}%),提交占用 ${p.avgCommitPercent}%。注意:内存占用高不一定是卡顿根因。`,
      },
      'rule.disk-active-high': {
        title: '磁盘活动持续偏高',
        desc: p => `磁盘平均活动 ${p.avgActivePercent}%,平均队列 ${p.avgQueue}。持续的磁盘压力会让桌面感觉冻结。`,
      },
      'rule.system-drive-low-space': {
        title: '系统盘剩余空间不足',
        desc: p => `${p.drive} 仅剩 ${p.freePercent}%(${p.freeGB} GB),会影响更新、页面文件与缓存。`,
      },
      'rule.no-major-issue': {
        title: '未发现明显瓶颈',
        desc: () => '本次采样窗口内 CPU 频率、内存、磁盘均未触发规则。如果卡顿是间歇性的,请在卡顿发生时运行深度扫描。',
      },
    },
    causes: {
      'cause.power-policy': { title: '电源策略限制', desc: '电源计划或处理器电源设置在压制 CPU 频率,可通过调整电源计划解决。' },
      'cause.firmware-adapter': { title: '固件/电源适配器限制', desc: '系统固件在限制处理器速度——常见于适配器不被识别、功率不足或老化,也可能是 BIOS 电源策略。' },
      'cause.vendor-manager': { title: '厂商电源管理软件', desc: 'Dell/Intel 的电源管理组件在接管调度策略,可能覆盖 Windows 默认行为。' },
      'cause.thermal': { title: '过热降频', desc: '温度或负载模式显示存在热限制的可能。' },
      'cause.battery': { title: '电池供电/老化', desc: '电池供电模式或电池老化在限制性能输出。' },
    },
    evidence: {
      'ev.avg-clock': p => `平均有效频率 ${p.avgMHz} MHz(基准 ${p.baseMHz} MHz)`,
      'ev.min-clock': p => `最低有效频率 ${p.minMHz} MHz`,
      'ev.low-freq-samples': p => `${p.percent}% 的样本低于基准频率的 ${p.thresholdPercent}%`,
      'ev.avg-load': p => `平均 CPU 负载 ${p.percent}%`,
      'ev.sample-count': p => `共 ${p.count} 个样本,间隔 ${p.intervalSec} 秒`,
      'ev.active-scheme': p => `当前电源计划:${p.name}${p.isPowerSaver ? '(节能)' : ''}`,
      'ev.max-proc-state': p => `最大处理器状态被限制为 ${p.percent}%(AC ${p.acPercent}% / DC ${p.dcPercent}%)`,
      'ev.boost-disabled': p => `处理器睿频已被禁用(AC 模式 ${p.acMode} / DC 模式 ${p.dcMode})`,
      'ev.throttle-events': p => `近 ${p.lookbackDays} 天有 ${p.count} 条「处理器速度被系统固件限制」事件,最近一次 ${p.lastTime}`,
      'ev.on-battery': () => '扫描时正在使用电池供电',
      'ev.vendor-service-running': p => `${p.displayName}(${p.name})正在运行`,
      'ev.thermal-temp': p => `温度传感器读数最高 ${p.maxC}°C`,
      'ev.thermal-pattern': p => `高负载(均值 ${p.avgLoadPercent}%)伴随 ${p.lowFreqSamplePercent}% 低频样本,符合热/功率限制模式`,
      'ev.battery-wear': p => `电池损耗 ${p.wearPercent}%${p.onAC ? '(当前接电源)' : '(当前用电池)'}`,
      'ev.mem-used': p => `内存占用均值 ${p.avgPercent}%,峰值 ${p.maxPercent}%`,
      'ev.commit': p => `提交占用均值 ${p.avgPercent}%`,
      'ev.disk-active': p => `磁盘活动均值 ${p.avgPercent}%`,
      'ev.disk-queue': p => `磁盘队列均值 ${p.avg}`,
      'ev.free-space': p => `${p.drive} 剩余 ${p.freePercent}%(${p.freeGB} GB)`,
    },
  },

  en: {
    ui: {
      appName: 'Diagnostic Studio',
      subtitle: 'Diagnostic Workbench',
      quickScan: 'Quick Scan (15s)',
      deepScan: '3-Min Deep Scan',
      scanning: 'Scanning',
      scanHint: 'Keep using the PC normally during the scan — the slow moments are exactly what sampling wants to catch',
      score: 'Health Score',
      primary: 'Primary Conclusion',
      attribution: 'Root Cause Attribution',
      attributionEmpty: 'Evidence is inconclusive — run the 3-minute deep scan while the slowdown is happening',
      findings: 'Findings',
      evidence: 'Evidence',
      samplingTitle: 'Effective CPU Clock',
      admin: 'Administrator',
      notAdmin: 'Not elevated (reduced visibility)',
      openLog: 'Open Logs',
      machine: 'Machine',
      noReport: 'Choose a scan mode to start diagnosing',
      collectorNotes: 'Collector Notes',
      severity: { good: 'Good', warning: 'Attention', critical: 'Critical' },
      findingSeverity: { info: 'Info', warning: 'Warning', critical: 'Critical' },
      confidence: { high: 'High confidence', medium: 'Medium confidence', low: 'Low confidence' },
      avgLoad: 'Avg Load',
      avgClock: 'Avg Clock',
      minClock: 'Min Clock',
      memUsed: 'Memory',
      diskActive: 'Disk Active',
    },
    rules: {
      'rule.cpu-freq-constrained': {
        title: 'CPU Frequency Constrained',
        desc: p => `The CPU averaged only ${p.avgFreqRatioPercent}% of base clock during sampling; ${p.lowFreqSamplePercent}% of samples were below 55% of base — the slowdown is more likely downclocking than surface resource usage.`,
      },
      'rule.power-saver-scheme': {
        title: 'Power Saver Plan Active',
        desc: p => `The active power plan "${p.schemeName}" limits processor responsiveness.`,
      },
      'rule.memory-pressure': {
        title: 'Memory Pressure',
        desc: p => `Memory averaged ${p.avgUsedPercent}% (peak ${p.maxUsedPercent}%), commit ${p.avgCommitPercent}%. Note: high memory usage is not necessarily the root cause of slowness.`,
      },
      'rule.disk-active-high': {
        title: 'Disk Activity Stayed High',
        desc: p => `Disk active averaged ${p.avgActivePercent}% with queue ${p.avgQueue}. Sustained disk pressure makes the desktop feel frozen.`,
      },
      'rule.system-drive-low-space': {
        title: 'System Drive Low on Space',
        desc: p => `${p.drive} has only ${p.freePercent}% (${p.freeGB} GB) free, affecting updates, paging, and caches.`,
      },
      'rule.no-major-issue': {
        title: 'No Major Bottleneck Detected',
        desc: () => 'CPU frequency, memory, and disk stayed within normal ranges in this sampling window. If the slowdown is intermittent, run the deep scan while it is happening.',
      },
    },
    causes: {
      'cause.power-policy': { title: 'Power Policy Limit', desc: 'The power plan or processor power settings are capping CPU frequency; adjusting the plan resolves this.' },
      'cause.firmware-adapter': { title: 'Firmware / Power Adapter Limit', desc: 'System firmware is limiting processor speed — typical of an unrecognized, undersized, or aging adapter, or BIOS power policy.' },
      'cause.vendor-manager': { title: 'Vendor Power Manager', desc: 'Dell/Intel power management components are overriding Windows default scheduling behavior.' },
      'cause.thermal': { title: 'Thermal Throttling', desc: 'Temperature or load patterns suggest a thermal limit.' },
      'cause.battery': { title: 'Battery Power / Wear', desc: 'Running on battery or battery degradation is limiting performance output.' },
    },
    evidence: {
      'ev.avg-clock': p => `Average effective clock ${p.avgMHz} MHz (base ${p.baseMHz} MHz)`,
      'ev.min-clock': p => `Minimum effective clock ${p.minMHz} MHz`,
      'ev.low-freq-samples': p => `${p.percent}% of samples below ${p.thresholdPercent}% of base clock`,
      'ev.avg-load': p => `Average CPU load ${p.percent}%`,
      'ev.sample-count': p => `${p.count} samples at ${p.intervalSec}s interval`,
      'ev.active-scheme': p => `Active power plan: ${p.name}${p.isPowerSaver ? ' (power saver)' : ''}`,
      'ev.max-proc-state': p => `Maximum processor state capped at ${p.percent}% (AC ${p.acPercent}% / DC ${p.dcPercent}%)`,
      'ev.boost-disabled': p => `Processor boost disabled (AC mode ${p.acMode} / DC mode ${p.dcMode})`,
      'ev.throttle-events': p => `${p.count} "processor speed limited by system firmware" events in the last ${p.lookbackDays} days, most recent ${p.lastTime}`,
      'ev.on-battery': () => 'Running on battery during the scan',
      'ev.vendor-service-running': p => `${p.displayName} (${p.name}) is running`,
      'ev.thermal-temp': p => `Thermal zone peak ${p.maxC}°C`,
      'ev.thermal-pattern': p => `High load (avg ${p.avgLoadPercent}%) with ${p.lowFreqSamplePercent}% low-frequency samples fits a thermal/power-limit pattern`,
      'ev.battery-wear': p => `Battery wear ${p.wearPercent}%${p.onAC ? ' (on AC now)' : ' (on battery now)'}`,
      'ev.mem-used': p => `Memory used avg ${p.avgPercent}%, peak ${p.maxPercent}%`,
      'ev.commit': p => `Commit charge avg ${p.avgPercent}%`,
      'ev.disk-active': p => `Disk active avg ${p.avgPercent}%`,
      'ev.disk-queue': p => `Disk queue avg ${p.avg}`,
      'ev.free-space': p => `${p.drive} free ${p.freePercent}% (${p.freeGB} GB)`,
    },
  },
};

export function t(lang) {
  return COPY[lang] || COPY.zh;
}

export function ruleText(lang, ruleId, params) {
  const r = t(lang).rules[ruleId];
  if (!r) return { title: ruleId, desc: JSON.stringify(params) };
  return { title: r.title, desc: r.desc(params || {}) };
}

export function causeText(lang, causeId) {
  return t(lang).causes[causeId] || { title: causeId, desc: '' };
}

export function evidenceText(lang, ev) {
  const f = t(lang).evidence[ev.evidenceId];
  if (!f) return `${ev.evidenceId} ${JSON.stringify(ev.params)}`;
  return f(ev.params || {});
}
