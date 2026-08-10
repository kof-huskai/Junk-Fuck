export namespace main {
	
	export class lastReportEntry {
	    report: report.Report;
	    at: string;
	
	    static createFrom(source: any = {}) {
	        return new lastReportEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.report = this.convertValues(source["report"], report.Report);
	        this.at = source["at"];
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

export namespace model {
	
	export class Candidate {
	    path: string;
	    name: string;
	    isDir: boolean;
	    category: string;
	    size: number;
	    reason: string;
	    protected: boolean;
	    scanSource: string;
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.isDir = source["isDir"];
	        this.category = source["category"];
	        this.size = source["size"];
	        this.reason = source["reason"];
	        this.protected = source["protected"];
	        this.scanSource = source["scanSource"];
	    }
	}
	export class Progress {
	    scanId: string;
	    scannedFiles: number;
	    candidates: number;
	    errors: number;
	    currentPath: string;
	    done: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Progress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scanId = source["scanId"];
	        this.scannedFiles = source["scannedFiles"];
	        this.candidates = source["candidates"];
	        this.errors = source["errors"];
	        this.currentPath = source["currentPath"];
	        this.done = source["done"];
	    }
	}
	export class ScanError {
	    path: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.error = source["error"];
	    }
	}

}

export namespace platform {
	
	export class Info {
	    os: string;
	    version: string;
	    isAdmin: boolean;
	    arch: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.version = source["version"];
	        this.isAdmin = source["isAdmin"];
	        this.arch = source["arch"];
	    }
	}

}

export namespace report {
	
	export class Item {
	    path: string;
	    name: string;
	    status: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	    }
	}
	export class Report {
	    dryRun: boolean;
	    deletedCount: number;
	    skippedCount: number;
	    failedCount: number;
	    bytesFreed: number;
	    items: Item[];
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dryRun = source["dryRun"];
	        this.deletedCount = source["deletedCount"];
	        this.skippedCount = source["skippedCount"];
	        this.failedCount = source["failedCount"];
	        this.bytesFreed = source["bytesFreed"];
	        this.items = this.convertValues(source["items"], Item);
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

