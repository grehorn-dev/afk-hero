export namespace domain {
	
	export class NumericRange {
	    mode: string;
	    value: number;
	    minVal: number;
	    maxVal: number;
	
	    static createFrom(source: any = {}) {
	        return new NumericRange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.value = source["value"];
	        this.minVal = source["minVal"];
	        this.maxVal = source["maxVal"];
	    }
	}
	export class RuntimeState {
	    state: string;
	    enabled: boolean;
	    progress: number;
	    statusKey: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.enabled = source["enabled"];
	        this.progress = source["progress"];
	        this.statusKey = source["statusKey"];
	    }
	}
	export class Settings {
	    enabled: boolean;
	    advanced: boolean;
	    logging: boolean;
	    language: string;
	    theme: string;
	    shape: string;
	    direction: string;
	    distance: NumericRange;
	    interval: NumericRange;
	    speed: NumericRange;
	    inactivity: NumericRange;
	    activationEnabled: boolean;
	    activationMode: string;
	    activationTimeout: number;
	    targetWindowTitle: string;
	    targetWindowClass: string;
	    targetProcessName: string;
	    activationTargetWindowOnly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.advanced = source["advanced"];
	        this.logging = source["logging"];
	        this.language = source["language"];
	        this.theme = source["theme"];
	        this.shape = source["shape"];
	        this.direction = source["direction"];
	        this.distance = this.convertValues(source["distance"], NumericRange);
	        this.interval = this.convertValues(source["interval"], NumericRange);
	        this.speed = this.convertValues(source["speed"], NumericRange);
	        this.inactivity = this.convertValues(source["inactivity"], NumericRange);
	        this.activationEnabled = source["activationEnabled"];
	        this.activationMode = source["activationMode"];
	        this.activationTimeout = source["activationTimeout"];
	        this.targetWindowTitle = source["targetWindowTitle"];
	        this.targetWindowClass = source["targetWindowClass"];
	        this.targetProcessName = source["targetProcessName"];
	        this.activationTargetWindowOnly = source["activationTargetWindowOnly"];
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
	export class WindowActivationState {
	    targetFound: boolean;
	    tracking: boolean;
	    progress: number;
	    processName: string;
	    windowTitle: string;
	
	    static createFrom(source: any = {}) {
	        return new WindowActivationState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetFound = source["targetFound"];
	        this.tracking = source["tracking"];
	        this.progress = source["progress"];
	        this.processName = source["processName"];
	        this.windowTitle = source["windowTitle"];
	    }
	}
	export class WindowInfo {
	    id: any;
	    title: string;
	    className: string;
	    processName: string;
	    autoCandidate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WindowInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.className = source["className"];
	        this.processName = source["processName"];
	        this.autoCandidate = source["autoCandidate"];
	    }
	}

}

export namespace i18n {
	
	export class Language {
	    code: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Language(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	    }
	}

}

