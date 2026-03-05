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
export class AxonToolkit {
    private axon: Axon;
    private sessionId: string;

    constructor(axonClient: Axon, sessionId: string = 'default') {
        this.axon = axonClient;
        this.sessionId = sessionId;
    }

    /**
     * Returns a list of tool definitions in OpenAI/LLM-friendly format.
     */
    getTools(): ToolDefinition[] {
        return [
            {
                name: 'navigate',
                description: 'Navigate to a URL and wait for the page to be ready.',
                parameters: {
                    type: 'object',
                    properties: {
                        url: { type: 'string', description: 'The destination URL' }
                    },
                    required: ['url']
                }
            },
            {
                name: 'snapshot',
                description: 'Get a compact semantic map of the current page. Uses 98% fewer tokens than raw HTML.',
                parameters: {
                    type: 'object',
                    properties: {}
                }
            },
            {
                name: 'smart_interact',
                description: 'The most powerful way to interact. Provide an intent (e.g. "search button") and an action.',
                parameters: {
                    type: 'object',
                    properties: {
                        intent: { type: 'string', description: 'Description of the element to interact with' },
                        action: { type: 'string', enum: ['click', 'fill', 'hover', 'press', 'select'] },
                        value: { type: 'string', description: 'Value for input or selection' }
                    },
                    required: ['intent', 'action']
                }
            },
            {
                name: 'wait_for_stability',
                description: 'Wait until the page is fully stable and animations are finished.',
                parameters: {
                    type: 'object',
                    properties: {}
                }
            }
        ];
    }

    /**
     * Executes a tool and returns the result as a string for the agent.
     */
    async runTool(toolName: string, args: any): Promise<string> {
        switch (toolName) {
            case 'navigate':
                await this.axon.navigate(this.sessionId, args.url);
                return `Successfully navigated to ${args.url}`;

            case 'snapshot':
                const snap = await this.axon.snapshot(this.sessionId);
                return `Page: ${snap.title}\nContent:\n${snap.content}`;

            case 'smart_interact':
                const res = await this.axon.smartInteract(
                    this.sessionId,
                    args.intent,
                    args.action,
                    args.value
                );
                if (res.requires_confirm) {
                    return `ACTION BLOCKED: This is an irreversible action. Set 'confirm: true' to proceed with: ${res.message}`;
                }
                return `Action '${args.action}' on '${args.intent}' was successful.`;

            case 'wait_for_stability':
                await this.axon.status(this.sessionId);
                return 'Page is now stable.';

            default:
                return `Tool ${toolName} not found.`;
        }
    }
}
