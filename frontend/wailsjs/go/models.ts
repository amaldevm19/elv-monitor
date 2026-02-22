export namespace main {
	
	export class ImportResult {
	    total: number;
	    added: number;
	    updated: number;
	    failed: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.failed = source["failed"];
	        this.errors = source["errors"];
	    }
	}

}

export namespace models {
	
	export class AuditLog {
	    id: string;
	    // Go type: time
	    ts: any;
	    userId?: string;
	    username?: string;
	    action: string;
	    target?: string;
	    detailsJson?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ts = this.convertValues(source["ts"], null);
	        this.userId = source["userId"];
	        this.username = source["username"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.detailsJson = source["detailsJson"];
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
	export class Device {
	    id: string;
	    name: string;
	    ip: string;
	    interval: number;
	    mode: string;
	    tcpPort: number;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.ip = source["ip"];
	        this.interval = source["interval"];
	        this.mode = source["mode"];
	        this.tcpPort = source["tcpPort"];
	        this.enabled = source["enabled"];
	    }
	}
	export class UserDTO {
	    id: string;
	    username: string;
	    role: string;
	
	    static createFrom(source: any = {}) {
	        return new UserDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.role = source["role"];
	    }
	}

}

