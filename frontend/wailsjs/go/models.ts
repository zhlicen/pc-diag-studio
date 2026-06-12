export namespace main {
	
	export class AppStatus {
	    version: string;
	    isAdmin: boolean;
	    logDir: string;
	
	    static createFrom(source: any = {}) {
	        return new AppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.isAdmin = source["isAdmin"];
	        this.logDir = source["logDir"];
	    }
	}

}

export namespace model {
	
	export class AttributionCandidate {
	    causeId: string;
	    confidence: string;
	    score: number;
	    evidence: Evidence[];
	
	    static createFrom(source: any = {}) {
	        return new AttributionCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.causeId = source["causeId"];
	        this.confidence = source["confidence"];
	        this.score = source["score"];
	        this.evidence = this.convertValues(source["evidence"], Evidence);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Evidence {
	    evidenceId: string;
	    params: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new Evidence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evidenceId = source["evidenceId"];
	        this.params = source["params"];
	    }
	}
	export class Finding {
	    ruleId: string;
	    severity: string;
	    params: Record<string, any>;
	    evidence: Evidence[];
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruleId = source["ruleId"];
	        this.severity = source["severity"];
	        this.params = source["params"];
	        this.evidence = this.convertValues(source["evidence"], Evidence);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AnalysisResult {
	    score: number;
	    severity: string;
	    primaryRuleId: string;
	    primaryParams: Record<string, any>;
	    findings: Finding[];
	    attribution: AttributionCandidate[];
	    categoryScores: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.score = source["score"];
	        this.severity = source["severity"];
	        this.primaryRuleId = source["primaryRuleId"];
	        this.primaryParams = source["primaryParams"];
	        this.findings = this.convertValues(source["findings"], Finding);
	        this.attribution = this.convertValues(source["attribution"], AttributionCandidate);
	        this.categoryScores = source["categoryScores"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CPUInfo {
	    name: string;
	    baseClockMHz: number;
	    cores: number;
	    logicalProcessors: number;
	
	    static createFrom(source: any = {}) {
	        return new CPUInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.baseClockMHz = source["baseClockMHz"];
	        this.cores = source["cores"];
	        this.logicalProcessors = source["logicalProcessors"];
	    }
	}
	export class ComputerInfo {
	    computerName: string;
	    manufacturer: string;
	    model: string;
	    osName: string;
	    osVersion: string;
	    osBuild: string;
	
	    static createFrom(source: any = {}) {
	        return new ComputerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.computerName = source["computerName"];
	        this.manufacturer = source["manufacturer"];
	        this.model = source["model"];
	        this.osName = source["osName"];
	        this.osVersion = source["osVersion"];
	        this.osBuild = source["osBuild"];
	    }
	}
	export class EventInfo {
	    timeCreated: string;
	    provider: string;
	    eventId: number;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new EventInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeCreated = source["timeCreated"];
	        this.provider = source["provider"];
	        this.eventId = source["eventId"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class InstalledApp {
	    name: string;
	    version: string;
	    publisher: string;
	    category: string;
	
	    static createFrom(source: any = {}) {
	        return new InstalledApp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.publisher = source["publisher"];
	        this.category = source["category"];
	    }
	}
	export class ScheduledTask {
	    name: string;
	    path: string;
	    state: string;
	
	    static createFrom(source: any = {}) {
	        return new ScheduledTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.state = source["state"];
	    }
	}
	export class StartupItem {
	    name: string;
	    command: string;
	    location: string;
	    user: string;
	    reviewWorthy: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StartupItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	        this.location = source["location"];
	        this.user = source["user"];
	        this.reviewWorthy = source["reviewWorthy"];
	    }
	}
	export class ProcessInfo {
	    name: string;
	    pid: number;
	    cpuPercent: number;
	    workingSetMB: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pid = source["pid"];
	        this.cpuPercent = source["cpuPercent"];
	        this.workingSetMB = source["workingSetMB"];
	    }
	}
	export class SamplingSummary {
	    sampleCount: number;
	    intervalSec: number;
	    avgEffectiveClockMHz: number;
	    minEffectiveClockMHz: number;
	    avgFreqRatioPercent: number;
	    lowFreqSamplePercent: number;
	    avgCpuLoadPercent: number;
	    maxCpuLoadPercent: number;
	    avgMemUsedPercent: number;
	    maxMemUsedPercent: number;
	    avgCommitPercent: number;
	    avgDiskActivePercent: number;
	    avgDiskQueue: number;
	
	    static createFrom(source: any = {}) {
	        return new SamplingSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sampleCount = source["sampleCount"];
	        this.intervalSec = source["intervalSec"];
	        this.avgEffectiveClockMHz = source["avgEffectiveClockMHz"];
	        this.minEffectiveClockMHz = source["minEffectiveClockMHz"];
	        this.avgFreqRatioPercent = source["avgFreqRatioPercent"];
	        this.lowFreqSamplePercent = source["lowFreqSamplePercent"];
	        this.avgCpuLoadPercent = source["avgCpuLoadPercent"];
	        this.maxCpuLoadPercent = source["maxCpuLoadPercent"];
	        this.avgMemUsedPercent = source["avgMemUsedPercent"];
	        this.maxMemUsedPercent = source["maxMemUsedPercent"];
	        this.avgCommitPercent = source["avgCommitPercent"];
	        this.avgDiskActivePercent = source["avgDiskActivePercent"];
	        this.avgDiskQueue = source["avgDiskQueue"];
	    }
	}
	export class Sample {
	    offsetSec: number;
	    cpuPerfPercent: number;
	    effectiveClockMHz: number;
	    cpuLoadPercent: number;
	    memUsedPercent: number;
	    commitPercent: number;
	    diskActivePercent: number;
	    diskQueue: number;
	
	    static createFrom(source: any = {}) {
	        return new Sample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.offsetSec = source["offsetSec"];
	        this.cpuPerfPercent = source["cpuPerfPercent"];
	        this.effectiveClockMHz = source["effectiveClockMHz"];
	        this.cpuLoadPercent = source["cpuLoadPercent"];
	        this.memUsedPercent = source["memUsedPercent"];
	        this.commitPercent = source["commitPercent"];
	        this.diskActivePercent = source["diskActivePercent"];
	        this.diskQueue = source["diskQueue"];
	    }
	}
	export class ServiceInfo {
	    name: string;
	    displayName: string;
	    state: string;
	    startMode: string;
	    vendorHint: string;
	    pathName: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.state = source["state"];
	        this.startMode = source["startMode"];
	        this.vendorHint = source["vendorHint"];
	        this.pathName = source["pathName"];
	    }
	}
	export class ThrottleEvent {
	    timeCreated: string;
	    eventId: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ThrottleEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeCreated = source["timeCreated"];
	        this.eventId = source["eventId"];
	        this.message = source["message"];
	    }
	}
	export class PowerStateInfo {
	    activeSchemeGuid: string;
	    activeSchemeName: string;
	    onAC: boolean;
	    acMaxProcessorState: number;
	    dcMaxProcessorState: number;
	    acBoostMode: number;
	    dcBoostMode: number;
	    batteryPresent: boolean;
	    batteryDesignedMWh: number;
	    batteryFullMWh: number;
	    batteryWearPercent: number;
	    thermalZoneMaxC: number;
	
	    static createFrom(source: any = {}) {
	        return new PowerStateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeSchemeGuid = source["activeSchemeGuid"];
	        this.activeSchemeName = source["activeSchemeName"];
	        this.onAC = source["onAC"];
	        this.acMaxProcessorState = source["acMaxProcessorState"];
	        this.dcMaxProcessorState = source["dcMaxProcessorState"];
	        this.acBoostMode = source["acBoostMode"];
	        this.dcBoostMode = source["dcBoostMode"];
	        this.batteryPresent = source["batteryPresent"];
	        this.batteryDesignedMWh = source["batteryDesignedMWh"];
	        this.batteryFullMWh = source["batteryFullMWh"];
	        this.batteryWearPercent = source["batteryWearPercent"];
	        this.thermalZoneMaxC = source["thermalZoneMaxC"];
	    }
	}
	export class DiskInfo {
	    drive: string;
	    label: string;
	    totalGB: number;
	    freeGB: number;
	    freePercent: number;
	    isSystem: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiskInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.drive = source["drive"];
	        this.label = source["label"];
	        this.totalGB = source["totalGB"];
	        this.freeGB = source["freeGB"];
	        this.freePercent = source["freePercent"];
	        this.isSystem = source["isSystem"];
	    }
	}
	export class MemoryInfo {
	    totalMB: number;
	
	    static createFrom(source: any = {}) {
	        return new MemoryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalMB = source["totalMB"];
	    }
	}
	export class DiagnosticReport {
	    schemaVersion: number;
	    generatedAt: string;
	    scanMode: string;
	    durationSec: number;
	    isAdmin: boolean;
	    computer: ComputerInfo;
	    cpu: CPUInfo;
	    memory: MemoryInfo;
	    disks: DiskInfo[];
	    power: PowerStateInfo;
	    throttleEvents: ThrottleEvent[];
	    vendorServices: ServiceInfo[];
	    samples: Sample[];
	    sampling: SamplingSummary;
	    processes: ProcessInfo[];
	    startupItems: StartupItem[];
	    scheduledTasks: ScheduledTask[];
	    installedApps: InstalledApp[];
	    systemEvents: EventInfo[];
	    analysis: AnalysisResult;
	    collectorNotes: string[];
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.generatedAt = source["generatedAt"];
	        this.scanMode = source["scanMode"];
	        this.durationSec = source["durationSec"];
	        this.isAdmin = source["isAdmin"];
	        this.computer = this.convertValues(source["computer"], ComputerInfo);
	        this.cpu = this.convertValues(source["cpu"], CPUInfo);
	        this.memory = this.convertValues(source["memory"], MemoryInfo);
	        this.disks = this.convertValues(source["disks"], DiskInfo);
	        this.power = this.convertValues(source["power"], PowerStateInfo);
	        this.throttleEvents = this.convertValues(source["throttleEvents"], ThrottleEvent);
	        this.vendorServices = this.convertValues(source["vendorServices"], ServiceInfo);
	        this.samples = this.convertValues(source["samples"], Sample);
	        this.sampling = this.convertValues(source["sampling"], SamplingSummary);
	        this.processes = this.convertValues(source["processes"], ProcessInfo);
	        this.startupItems = this.convertValues(source["startupItems"], StartupItem);
	        this.scheduledTasks = this.convertValues(source["scheduledTasks"], ScheduledTask);
	        this.installedApps = this.convertValues(source["installedApps"], InstalledApp);
	        this.systemEvents = this.convertValues(source["systemEvents"], EventInfo);
	        this.analysis = this.convertValues(source["analysis"], AnalysisResult);
	        this.collectorNotes = source["collectorNotes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	
	
	
	
	

}

