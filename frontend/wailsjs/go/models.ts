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
	
	    static createFrom(source: any = {}) {
	        return new OpenRouterConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.openrouter_api_key = source["openrouter_api_key"];
	        this.chat_model_id = source["chat_model_id"];
	        this.story_processing_model_id = source["story_processing_model_id"];
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

}

