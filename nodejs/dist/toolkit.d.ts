import { Axon } from './index.js';
export interface ToolDefinition {
    name: string;
    description: string;
    parameters: {
        type: 'object';
        properties: Record<string, any>;
        required?: string[];
    };
}
/**
 * A ready-to-use sensory kit for AI agents (Vamora, LangChain, etc.)
 */
export declare class AxonToolkit {
    private axon;
    private sessionId;
    constructor(axonClient: Axon, sessionId?: string);
    /**
     * Returns a list of tool definitions in OpenAI/LLM-friendly format.
     */
    getTools(): ToolDefinition[];
    /**
     * Executes a tool and returns the result as a string for the agent.
     */
    runTool(toolName: string, args: any): Promise<string>;
}
//# sourceMappingURL=toolkit.d.ts.map