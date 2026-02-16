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

}

