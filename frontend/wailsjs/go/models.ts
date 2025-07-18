export namespace database {
	
	export class CodexEntry {
	    id: number;
	    name: string;
	    type: string;
	    content: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CodexEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.content = source["content"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

export namespace llm {
	
	export class OpenRouterConfig {
	    openrouter_api_key: string;
	    chat_model_id?: string;
	    story_processing_model_id?: string;
	    gemini_api_key?: string;
	    active_mode?: string;
	    openai_api_key?: string;
	    local_embedding_model_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenRouterConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.openrouter_api_key = source["openrouter_api_key"];
	        this.chat_model_id = source["chat_model_id"];
	        this.story_processing_model_id = source["story_processing_model_id"];
	        this.gemini_api_key = source["gemini_api_key"];
	        this.active_mode = source["active_mode"];
	        this.openai_api_key = source["openai_api_key"];
	        this.local_embedding_model_name = source["local_embedding_model_name"];
	    }
	}
	export class OpenRouterModel {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenRouterModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

export namespace main {
	
	export class ChatMessage {
	    sender: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sender = source["sender"];
	        this.text = source["text"];
	    }
	}
	export class LibraryItem {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	    // Go type: time
	    modTime: any;
	    children: LibraryItem[];
	
	    static createFrom(source: any = {}) {
	        return new LibraryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.modTime = this.convertValues(source["modTime"], null);
	        this.children = this.convertValues(source["children"], LibraryItem);
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
	export class ProcessStoryResult {
	    newEntries: database.CodexEntry[];
	    updatedEntries: database.CodexEntry[];
	    existingEntries: database.CodexEntry[];
	
	    static createFrom(source: any = {}) {
	        return new ProcessStoryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.newEntries = this.convertValues(source["newEntries"], database.CodexEntry);
	        this.updatedEntries = this.convertValues(source["updatedEntries"], database.CodexEntry);
	        this.existingEntries = this.convertValues(source["existingEntries"], database.CodexEntry);
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

