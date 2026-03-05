/**
 * Axon Node.js SDK - Browser automation client
 */
import type { SessionInfo, CreateSessionResponse, SnapshotResponse, ActionResponse, NavigateResponse, ReplayResponse, SessionList, AxonOptions } from './types.js';
import { EngineOptions } from './engine.js';
import { AxonToolkit } from './toolkit.js';
export { SessionInfo, CreateSessionResponse, SnapshotResponse, ActionResponse, NavigateResponse, ReplayResponse } from './types.js';
export { AxonToolkit };
/**
 * Axon API Error
 */
export declare class AxonError extends Error {
    statusCode: number;
    constructor(message: string, statusCode?: number);
}
/**
 * Axon browser automation client
 *
 * @example
 * ```typescript
 * import { Axon } from '@axon/browser';
 *
 * const axon = new Axon('http://localhost:8020/api/v1');
 *
 * // Create a session
 * const session = await axon.createSession('mysession');
 *
 * // Navigate
 * await axon.navigate('mysession', 'https://github.com');
 *
 * // Get snapshot
 * const snapshot = await axon.snapshot('mysession');
 * console.log(snapshot.title);
 *
 * // Click
 * await axon.click('mysession', 'e1');
 * ```
 */
export declare class Axon {
    private apiUrl;
    private headers;
    private engine;
    /**
     * Create a new Axon client
     *
     * @param options - Client configuration
     */
    constructor(options?: AxonOptions & EngineOptions & {
        startEngine?: boolean;
    });
    /**
     * Start the underlying Axon engine if configured
     */
    startEngine(): Promise<void>;
    /**
     * Stop the underlying Axon engine if configured
     */
    stopEngine(): void;
    /**
     * Make an API request
     */
    private request;
    /**
     * Create a new browser session
     *
     * @param sessionId - Unique identifier for the session
     * @param profile - Optional browser profile name
     */
    createSession(sessionId: string, profile?: string): Promise<CreateSessionResponse>;
    /**
     * Get session information
     *
     * @param sessionId - The session ID
     */
    getSession(sessionId: string): Promise<SessionInfo>;
    /**
     * List all active sessions
     */
    listSessions(): Promise<SessionList>;
    /**
     * Delete a session
     *
     * @param sessionId - The session ID to delete
     */
    deleteSession(sessionId: string): Promise<void>;
    /**
     * Navigate to a URL
     *
     * @param sessionId - The session ID
     * @param url - The URL to navigate to
     */
    navigate(sessionId: string, url: string, waitUntil?: 'none' | 'load' | 'domcontentloaded' | 'networkidle'): Promise<NavigateResponse>;
    /**
     * Get a snapshot of the current page
     *
     * @param sessionId - The session ID
     * @param ref - Optional element reference to focus on
     */
    snapshot(sessionId: string, ref?: string): Promise<SnapshotResponse>;
    /**
     * Perform an action on an element
     *
     * @param sessionId - The session ID
     * @param action - Action to perform
     * @param ref - Element reference ID
     * @param value - Optional value for fill/select actions
     * @param confirm - Confirm irreversible action
     */
    act(sessionId: string, action: string, ref: string, value?: string, confirm?: boolean): Promise<ActionResponse>;
    /**
     * Click an element
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     */
    click(sessionId: string, ref: string): Promise<ActionResponse>;
    /**
     * Fill an input field
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     * @param value - Value to fill
     */
    fill(sessionId: string, ref: string, value: string): Promise<ActionResponse>;
    /**
     * Fill an input field using a secret from the Intelligence Vault
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     * @param secretName - Name of the secret in the vault
     * @param field - Field name to inject (username, password, value)
     */
    vaultFill(sessionId: string, ref: string, secretName: string, field?: string): Promise<ActionResponse>;
    /**
     * Add a secret to the Intelligence Vault
     *
     * @param name - Friendly name for the secret
     * @param value - Secret value (for generic secrets)
     * @param url - Domain/URL the secret is bound to
     * @param options - Additional fields (username, password, labels)
     */
    addSecret(name: string, value: string, url: string, options?: {
        username?: string;
        password?: string;
        labels?: string[];
    }): Promise<boolean>;
    /**
     * Hover over an element
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     */
    hover(sessionId: string, ref: string): Promise<ActionResponse>;
    /**
     * Select an option
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     * @param value - Value to select
     */
    select(sessionId: string, ref: string, value: string): Promise<ActionResponse>;
    /**
     * Check a checkbox
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     */
    check(sessionId: string, ref: string): Promise<ActionResponse>;
    /**
     * Uncheck a checkbox
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     */
    uncheck(sessionId: string, ref: string): Promise<ActionResponse>;
    /**
     * Find an element by description and perform an action
     *
     * @param sessionId - The session ID
     * @param action - Action to perform
     * @param description - Semantic description of the element
     * @param value - Optional value for the action
     */
    findAndAct(sessionId: string, action: string, intent: string, value?: string): Promise<ActionResponse>;
    /**
     * Get a replay of the session history
     *
     * @param sessionId - The session ID
     */
    replay(sessionId: string): Promise<ReplayResponse>;
    /**
     * Get session status
     *
     * @param sessionId - The session ID
     */
    status(sessionId: string): Promise<Record<string, unknown>>;
    /**
     * High-level interaction tool for agents.
     * Automatically resolves intent and handles safety checks.
     */
    smartInteract(sessionId: string, intent: string, action?: string, value?: string): Promise<ActionResponse>;
}
export default Axon;
//# sourceMappingURL=index.d.ts.map