export namespace main {
	
	export class RunningApp {
	    pid: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new RunningApp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.name = source["name"];
	    }
	}

}

