import { spawn, ChildProcess } from 'child_process';
import { fetch } from 'undici';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import fs from 'fs';

const __dirname = dirname(fileURLToPath(import.meta.url));

export interface EngineOptions {
    binaryPath?: string;
    configPath?: string;
    port?: number;
    host?: string;
}

/**
 * Manages the Axon browser engine process.
 */
export class AxonEngine {
    private process: ChildProcess | null = null;
    private binaryPath: string;
    private configPath?: string;
    private port: number;
    private host: string;

    constructor(options: EngineOptions = {}) {
        this.port = options.port || 8020;
        this.host = options.host || '127.0.0.1';
        this.configPath = options.configPath;

        if (options.binaryPath) {
            this.binaryPath = options.binaryPath;
        } else {
            // Look for axon binary in the package or current directory
            const potentialPaths = [
                join(__dirname, '..', 'bin', 'axon.exe'),
                join(__dirname, '..', '..', 'bin', 'axon.exe'),
                join(process.cwd(), 'axon.exe'),
                join(process.cwd(), 'bin', 'axon.exe'),
            ];

            this.binaryPath = potentialPaths.find(p => fs.existsSync(p)) || 'axon.exe';
        }
    }

    /**
     * Check if the engine is already running
     */
    async isRunning(): Promise<boolean> {
        try {
            // Use undici to check if port is responsive
            const response = await fetch(`http://${this.host}:${this.port}/api/v1/sessions`).catch(() => null);
            return response !== null && response.status < 500;
        } catch {
            return false;
        }
    }

    /**
     * Start the engine
     */
    async start(timeout: number = 15000): Promise<void> {
        if (await this.isRunning()) {
            console.log(`Axon engine already running on ${this.host}:${this.port}`);
            return;
        }

        if (!fs.existsSync(this.binaryPath)) {
            throw new Error(`Axon binary not found at ${this.binaryPath}. Please provide a valid path.`);
        }

        const args: string[] = [];
        if (this.configPath) {
            args.push('--config', this.configPath);
        }

        console.log(`Starting Axon engine: ${this.binaryPath} ${args.join(' ')}`);

        this.process = spawn(this.binaryPath, args, {
            stdio: 'ignore',
            detached: true,
            windowsHide: true,
        });

        this.process.unref();

        // Wait for engine to be ready
        const startTime = Date.now();
        while (Date.now() - startTime < timeout) {
            if (await this.isRunning()) {
                console.log('Axon engine started successfully.');
                return;
            }
            await new Promise(resolve => setTimeout(resolve, 500));
        }

        this.stop();
        throw new Error('Timed out waiting for Axon engine to start.');
    }

    /**
     * Stop the engine
     */
    stop(): void {
        if (this.process) {
            console.log('Stopping Axon engine process...');
            this.process.kill();
            this.process = null;
        }
    }
}
