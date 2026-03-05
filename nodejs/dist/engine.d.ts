export interface EngineOptions {
    binaryPath?: string;
    configPath?: string;
    port?: number;
    host?: string;
}
/**
 * Manages the Axon browser engine process.
 */
export declare class AxonEngine {
    private process;
    private binaryPath;
    private configPath?;
    private port;
    private host;
    constructor(options?: EngineOptions);
    /**
     * Check if the engine is already running
     */
    isRunning(): Promise<boolean>;
    /**
     * Start the engine
     */
    start(timeout?: number): Promise<void>;
    /**
     * Stop the engine
     */
    stop(): void;
}
//# sourceMappingURL=engine.d.ts.map