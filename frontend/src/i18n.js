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
      tabs: { overview: '概览', actions: '优化', processes: '进程', startup: '启动项', services: '服务', software: '软件', events: '事件' },
      act: {
        run: '执行', confirmTitle: '确认执行此操作?', confirm: '确认执行', cancel: '取消',
        running: '执行中…', success: '执行成功', failed: '执行失败',
        recommendOnly: '仅建议(不会自动执行)',
        risk: { safe: '安全', review: '需评估', caution: '谨慎', high: '高风险' },
        rollbackTitle: '服务回滚记录',
        rollback: '一键还原',
        rolledBack: '已还原',
        noActions: '本次诊断未产生可执行的优化动作。',
        noRollback: '暂无回滚记录。',
        rollbackHint: '禁用服务前会先把原启动方式写入回滚记录(写入失败则不执行),可随时在此还原。',
        irreversible: '注意:此操作不可恢复。',
        freed: p => `已清理 ${p.freedMB} MB(${p.files} 个文件)`,
        recPrev: '原状态',
      },
      charts: { load: 'CPU 负载 (%)', mem: '内存占用 (%)', disk: '磁盘活动 (%)' },
      th: {
        procName: '进程', pid: 'PID', cpu: 'CPU', mem: '内存',
        name: '名称', command: '命令', location: '位置', reviewWorthy: '建议检查',
        taskPath: '任务路径', state: '状态',
        displayName: '显示名', startMode: '启动方式', hint: '分类',
        version: '版本', publisher: '发布者', category: '类别',
        time: '时间', level: '级别', provider: '来源', eventId: '事件 ID', message: '消息',
      },
      yes: '是',
      empty: '无数据',
      startupTasksTitle: '计划任务(非系统)',
      utilityCategories: { browser: '浏览器', archive: '压缩工具', assistant: '助手/管家类' },
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
      'rule.startup-load': {
        title: '启动项与后台任务偏多',
        desc: p => `检测到 ${p.startupCount} 个值得检查的启动项和 ${p.taskCount} 个第三方计划任务,共 ${p.count} 项。过多的更新器和助手会拖慢登录和日常响应。`,
      },
      'rule.vendor-services': {
        title: 'Dell/Intel 后台服务',
        desc: p => p.busyCount > 0
          ? `${p.count} 个 Dell/Intel 服务在运行,其中 ${p.busyCount} 个在扫描期间有实际资源消耗(见证据)。`
          : `${p.count} 个 Dell/Intel 服务在运行。数量本身属于正常背景信息,扫描期间未观测到明显资源消耗;如需精简可在优化页逐项处理。`,
      },
      'rule.duplicate-utilities': {
        title: '同类工具软件重复安装',
        desc: p => `检测到 ${p.count} 款同类软件(${p.category})。重复安装通常意味着重复的后台更新器和启动项。`,
      },
      'rule.no-major-issue': {
        title: '未发现明显瓶颈',
        desc: () => '本次采样窗口内 CPU 频率、内存、磁盘均未触发规则。如果卡顿是间歇性的,请在卡顿发生时运行深度扫描。',
      },
    },
    actions: {
      'action.power-high-performance': {
        title: () => '切换到高性能电源计划',
        desc: p => `当前电源计划为「${p.currentScheme}」。切换到高性能计划可解除调度限制,随时可在 Windows 设置中改回。`,
      },
      'action.reset-max-proc-state': {
        title: () => '恢复最大处理器状态到 100%',
        desc: p => `当前限制:AC ${p.acPercent}% / DC ${p.dcPercent}%。低于 100% 会直接压制 CPU 频率上限。`,
      },
      'action.clean-temp': {
        title: () => '清理临时文件',
        desc: () => '删除用户与系统临时目录中未被占用的文件,释放系统盘空间。',
      },
      'action.disable-service': {
        title: p => `禁用服务:${p.displayName}`,
        desc: p => {
          const power = ['dell-optimizer', 'dell-power-manager', 'intel-dtt'].includes(p.hint);
          return `服务名 ${p.serviceName}。禁用前会先保存原启动方式到回滚记录,可一键还原。${power ? '该服务可能影响电源调度行为。' : '禁用可减少厂商后台负载。'}`;
        },
      },
      'action.uninstall-recommendation': {
        title: p => `建议卸载:${p.displayName}`,
        desc: () => '如确认不需要该组件,请在 设置 > 应用 > 安装的应用 中手动卸载。本工具不会自动卸载任何软件。',
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
      'ev.throttle-events': p => `近 ${p.lookbackDays} 天有 ${p.count} 条「处理器速度被系统固件限制」事件,最近一次 ${p.lastTime}${p.recent ? '(48 小时内仍在发生)' : ''}`,
      'ev.policy-consistency': p => `观测到的平均频率比 ${p.observedPercent}% 与策略上限 ${p.capPercent}% 吻合——该设置可直接解释降频`,
      'ev.startup-count': p => `${p.count} 个值得检查的启动项`,
      'ev.task-count': p => `${p.count} 个第三方计划任务`,
      'ev.vendor-service-count': p => `${p.count} 个 Dell/Intel 服务正在运行`,
      'ev.vendor-busy': p => `${p.displayName}(进程 ${p.process})CPU ${p.cpuPercent}%、内存 ${p.memMB} MB`,
      'ev.duplicate-apps': p => `同类软件:${p.names}`,
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
      tabs: { overview: 'Overview', actions: 'Actions', processes: 'Processes', startup: 'Startup', services: 'Services', software: 'Software', events: 'Events' },
      act: {
        run: 'Run', confirmTitle: 'Run this action?', confirm: 'Run', cancel: 'Cancel',
        running: 'Running…', success: 'Succeeded', failed: 'Failed',
        recommendOnly: 'Recommendation only (never auto-executed)',
        risk: { safe: 'Safe', review: 'Review', caution: 'Caution', high: 'High risk' },
        rollbackTitle: 'Service Rollback Records',
        rollback: 'Restore',
        rolledBack: 'Restored',
        noActions: 'This diagnosis produced no executable optimization actions.',
        noRollback: 'No rollback records yet.',
        rollbackHint: 'Before disabling a service the previous startup mode is written to a rollback record (the action aborts if the write fails). Restore any time from here.',
        irreversible: 'Note: this operation cannot be undone.',
        freed: p => `Freed ${p.freedMB} MB (${p.files} files)`,
        recPrev: 'Previous',
      },
      charts: { load: 'CPU Load (%)', mem: 'Memory Used (%)', disk: 'Disk Active (%)' },
      th: {
        procName: 'Process', pid: 'PID', cpu: 'CPU', mem: 'Memory',
        name: 'Name', command: 'Command', location: 'Location', reviewWorthy: 'Review',
        taskPath: 'Task Path', state: 'State',
        displayName: 'Display Name', startMode: 'Start Mode', hint: 'Category',
        version: 'Version', publisher: 'Publisher', category: 'Category',
        time: 'Time', level: 'Level', provider: 'Provider', eventId: 'Event ID', message: 'Message',
      },
      yes: 'Yes',
      empty: 'No data',
      startupTasksTitle: 'Scheduled Tasks (non-system)',
      utilityCategories: { browser: 'Browser', archive: 'Archiver', assistant: 'Assistant/Manager' },
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
      'rule.startup-load': {
        title: 'Heavy Startup and Background Task Load',
        desc: p => `Found ${p.startupCount} review-worthy startup entries and ${p.taskCount} third-party scheduled tasks (${p.count} total). Updaters and assistants slow login and day-to-day responsiveness.`,
      },
      'rule.vendor-services': {
        title: 'Dell/Intel Background Services',
        desc: p => p.busyCount > 0
          ? `${p.count} Dell/Intel services are running; ${p.busyCount} consumed measurable resources during the scan (see evidence).`
          : `${p.count} Dell/Intel services are running. The count itself is normal background context — no significant resource usage was observed during the scan. Trim individually from the Actions tab if desired.`,
      },
      'rule.duplicate-utilities': {
        title: 'Duplicate Utility Software',
        desc: p => `${p.count} apps of the same kind (${p.category}) are installed. Duplicates usually mean duplicate background updaters and startup entries.`,
      },
      'rule.no-major-issue': {
        title: 'No Major Bottleneck Detected',
        desc: () => 'CPU frequency, memory, and disk stayed within normal ranges in this sampling window. If the slowdown is intermittent, run the deep scan while it is happening.',
      },
    },
    actions: {
      'action.power-high-performance': {
        title: () => 'Switch to High Performance Power Plan',
        desc: p => `Active plan is "${p.currentScheme}". High performance removes scheduling limits; revert any time in Windows Settings.`,
      },
      'action.reset-max-proc-state': {
        title: () => 'Reset Maximum Processor State to 100%',
        desc: p => `Current cap: AC ${p.acPercent}% / DC ${p.dcPercent}%. Anything below 100% directly caps CPU frequency.`,
      },
      'action.clean-temp': {
        title: () => 'Clean Temporary Files',
        desc: () => 'Deletes unlocked files under the user and system temp directories to free system-drive space.',
      },
      'action.disable-service': {
        title: p => `Disable Service: ${p.displayName}`,
        desc: p => {
          const power = ['dell-optimizer', 'dell-power-manager', 'intel-dtt'].includes(p.hint);
          return `Service ${p.serviceName}. The previous startup mode is saved to a rollback record first; one-click restore available. ${power ? 'This service may influence power scheduling.' : 'Disabling reduces vendor background load.'}`;
        },
      },
      'action.uninstall-recommendation': {
        title: p => `Consider Uninstalling: ${p.displayName}`,
        desc: () => 'If you confirm this component is unnecessary, uninstall it manually via Settings > Apps. This tool never uninstalls software automatically.',
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
      'ev.throttle-events': p => `${p.count} "processor speed limited by system firmware" events in the last ${p.lookbackDays} days, most recent ${p.lastTime}${p.recent ? ' (still occurring within 48h)' : ''}`,
      'ev.policy-consistency': p => `Observed average frequency ratio ${p.observedPercent}% matches the policy cap of ${p.capPercent}% — the setting directly explains the downclocking`,
      'ev.startup-count': p => `${p.count} review-worthy startup entries`,
      'ev.task-count': p => `${p.count} third-party scheduled tasks`,
      'ev.vendor-service-count': p => `${p.count} Dell/Intel services running`,
      'ev.vendor-busy': p => `${p.displayName} (process ${p.process}) CPU ${p.cpuPercent}%, memory ${p.memMB} MB`,
      'ev.duplicate-apps': p => `Same-category apps: ${p.names}`,
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
  let p = params || {};
  // Localize category codes embedded in params (duplicate-utilities rule).
  const cats = t(lang).ui.utilityCategories;
  if (p.category && cats && cats[p.category]) p = { ...p, category: cats[p.category] };
  return { title: r.title, desc: r.desc(p) };
}

export function causeText(lang, causeId) {
  return t(lang).causes[causeId] || { title: causeId, desc: '' };
}

export function actionText(lang, actionId, params) {
  const a = t(lang).actions[actionId];
  if (!a) return { title: actionId, desc: JSON.stringify(params) };
  return { title: a.title(params || {}), desc: a.desc(params || {}) };
}

export function evidenceText(lang, ev) {
  const f = t(lang).evidence[ev.evidenceId];
  if (!f) return `${ev.evidenceId} ${JSON.stringify(ev.params)}`;
  return f(ev.params || {});
}
