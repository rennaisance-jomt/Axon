/**
 * Axon Node.js SDK - Browser automation client
 */
import { fetch, Headers } from 'undici';
import { AxonEngine } from './engine.js';
import { AxonToolkit } from './toolkit.js';
export { AxonToolkit };
/**
 * Axon API Error
 */
export class AxonError extends Error {
    statusCode;
    constructor(message, statusCode = 0) {
        super(message);
        this.name = 'AxonError';
        this.statusCode = statusCode;
    }
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
export class Axon {
    apiUrl;
    headers;
    engine = null;
    /**
     * Create a new Axon client
     *
     * @param options - Client configuration
     */
    constructor(options = {}) {
        this.apiUrl = options.apiUrl || process.env.AXON_API_URL || 'http://localhost:8020/api/v1';
        this.headers = new Headers({
            'Content-Type': 'application/json',
        });
        if (options.startEngine) {
            let port = 8020;
            try {
                const url = new URL(this.apiUrl);
                port = parseInt(url.port) || 80;
            }
            catch (e) { }
            this.engine = new AxonEngine({
                binaryPath: options.binaryPath,
                configPath: options.configPath,
                port: port,
            });
        }
    }
    /**
     * Start the underlying Axon engine if configured
     */
    async startEngine() {
        if (this.engine) {
            await this.engine.start();
        }
        else {
            throw new Error('Engine management not enabled. Initialize with startEngine: true');
        }
    }
    /**
     * Stop the underlying Axon engine if configured
     */
    stopEngine() {
        if (this.engine) {
            this.engine.stop();
        }
    }
    /**
     * Make an API request
     */
    async request(method, path, body) {
        const url = `${this.apiUrl}${path}`;
        const response = await fetch(url, {
            method,
            headers: this.headers,
            body: body ? JSON.stringify(body) : undefined,
        });
        if (response.status >= 400) {
            const errorText = await response.text();
            throw new AxonError(`API error (${response.status}): ${errorText}`, response.status);
        }
        if (response.status === 204) {
            return {};
        }
        return response.json();
    }
    // ================== Session Management ==================
    /**
     * Create a new browser session
     *
     * @param sessionId - Unique identifier for the session
     * @param profile - Optional browser profile name
     */
    async createSession(sessionId, profile) {
        const data = { id: sessionId };
        if (profile) {
            data.profile = profile;
        }
        return this.request('POST', '/sessions', data);
    }
    /**
     * Get session information
     *
     * @param sessionId - The session ID
     */
    async getSession(sessionId) {
        return this.request('GET', `/sessions/${sessionId}`);
    }
    /**
     * List all active sessions
     */
    async listSessions() {
        return this.request('GET', '/sessions');
    }
    /**
     * Delete a session
     *
     * @param sessionId - The session ID to delete
     */
    async deleteSession(sessionId) {
        await this.request('DELETE', `/sessions/${sessionId}`);
    }
    // ================== Navigation ==================
    /**
     * Navigate to a URL
     *
     * @param sessionId - The session ID
     * @param url - The URL to navigate to
     */
    async navigate(sessionId, url, waitUntil = 'load') {
        return this.request('POST', `/sessions/${sessionId}/navigate`, {
            url,
            wait_until: waitUntil,
        });
    }
    // ================== Snapshots ==================
    /**
     * Get a snapshot of the current page
     *
     * @param sessionId - The session ID
     * @param ref - Optional element reference to focus on
     */
    async snapshot(sessionId, ref) {
        const data = {};
        if (ref) {
            data.ref = ref;
        }
        return this.request('POST', `/sessions/${sessionId}/snapshot`, data);
    }
    // ================== Actions ==================
    /**
     * Perform an action on an element
     *
     * @param sessionId - The session ID
     * @param action - Action to perform
     * @param ref - Element reference ID
     * @param value - Optional value for fill/select actions
     * @param confirm - Confirm irreversible action
     */
    async act(sessionId, action, ref, value, confirm = false) {
        const data = {
            action,
            ref,
            confirm,
        };
        if (value !== undefined) {
            data.value = value;
        }
        return this.request('POST', `/sessions/${sessionId}/act`, data);
    }
    /**
     * Click an element
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     */
    async click(sessionId, ref) {
        return this.act(sessionId, 'click', ref);
    }
    /**
     * Fill an input field
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     * @param value - Value to fill
     */
    async fill(sessionId, ref, value) {
        return this.act(sessionId, 'fill', ref, value);
    }
    /**
     * Fill an input field using a secret from the Intelligence Vault
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     * @param secretName - Name of the secret in the vault
     * @param field - Field name to inject (username, password, value)
     */
    async vaultFill(sessionId, ref, secretName, field = 'password') {
        const vaultRef = `@vault:${secretName}:${field}`;
        return this.fill(sessionId, ref, vaultRef);
    }
    // ================== Vault Management ==================
    /**
     * Add a secret to the Intelligence Vault
     *
     * @param name - Friendly name for the secret
     * @param value - Secret value (for generic secrets)
     * @param url - Domain/URL the secret is bound to
     * @param options - Additional fields (username, password, labels)
     */
    async addSecret(name, value, url, options = {}) {
        const data = {
            name,
            value,
            url,
            ...options,
        };
        const result = await this.request('POST', '/vault/secrets', data);
        return result.success;
    }
    /**
     * Hover over an element
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     */
    async hover(sessionId, ref) {
        return this.act(sessionId, 'hover', ref);
    }
    /**
     * Select an option
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     * @param value - Value to select
     */
    async select(sessionId, ref, value) {
        return this.act(sessionId, 'select', ref, value);
    }
    /**
     * Check a checkbox
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     */
    async check(sessionId, ref) {
        return this.act(sessionId, 'check', ref);
    }
    /**
     * Uncheck a checkbox
     *
     * @param sessionId - The session ID
     * @param ref - Element reference ID
     */
    async uncheck(sessionId, ref) {
        return this.act(sessionId, 'uncheck', ref);
    }
    // ================== Find and Act ==================
    /**
     * Find an element by description and perform an action
     *
     * @param sessionId - The session ID
     * @param action - Action to perform
     * @param description - Semantic description of the element
     * @param value - Optional value for the action
     */
    async findAndAct(sessionId, action, intent, value) {
        const data = { action, intent };
        if (value !== undefined) {
            data.value = value;
        }
        return this.request('POST', `/sessions/${sessionId}/find_and_act`, data);
    }
    // ================== Replay ==================
    /**
     * Get a replay of the session history
     *
     * @param sessionId - The session ID
     */
    async replay(sessionId) {
        return this.request('GET', `/sessions/${sessionId}/replay`);
    }
    // ================== Status ==================
    /**
     * Get session status
     *
     * @param sessionId - The session ID
     */
    async status(sessionId) {
        return this.request('GET', `/sessions/${sessionId}/status`);
    }
    // ================== Smart Agent Tools ==================
    /**
     * High-level interaction tool for agents.
     * Automatically resolves intent and handles safety checks.
     */
    async smartInteract(sessionId, intent, action = 'click', value) {
        return this.findAndAct(sessionId, action, intent, value);
    }
}
export default Axon;
