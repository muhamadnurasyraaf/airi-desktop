export namespace main {
	
	export class ConversationMessage {
	    role: string;
	    content: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new ConversationMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	    }
	}
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

