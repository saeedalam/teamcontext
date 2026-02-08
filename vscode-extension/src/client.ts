import * as cp from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

/**
 * TeamContextClient communicates with the teamcontext binary via JSON-RPC over stdio.
 * It spawns the MCP server as a child process and sends tool calls.
 */
export class TeamContextClient {
    private process: cp.ChildProcess | null = null;
    private requestId = 0;
    private pendingRequests = new Map<number, { resolve: (v: any) => void; reject: (e: any) => void }>();
    private buffer = '';

    constructor(private workspaceRoot: string) {}

    private ensureProcess(): cp.ChildProcess {
        if (this.process && !this.process.killed) {
            return this.process;
        }

        const binaryPath = this.findBinary();
        if (!binaryPath) {
            throw new Error('teamcontext binary not found in PATH');
        }

        this.process = cp.spawn(binaryPath, ['serve', this.workspaceRoot], {
            stdio: ['pipe', 'pipe', 'pipe'],
        });

        this.process.stdout?.on('data', (data: Buffer) => {
            this.buffer += data.toString();
            this.processBuffer();
        });

        this.process.on('exit', () => {
            this.process = null;
        });

        // Send initialize
        this.sendRaw({
            jsonrpc: '2.0',
            id: this.requestId++,
            method: 'initialize',
            params: {
                protocolVersion: '2024-11-05',
                capabilities: {},
                clientInfo: { name: 'vscode-teamcontext', version: '0.1.0' },
            },
        });

        return this.process;
    }

    async callTool(name: string, args: Record<string, any> = {}): Promise<any> {
        this.ensureProcess();

        const id = this.requestId++;
        const request = {
            jsonrpc: '2.0',
            id,
            method: 'tools/call',
            params: { name, arguments: args },
        };

        return new Promise((resolve, reject) => {
            this.pendingRequests.set(id, { resolve, reject });
            this.sendRaw(request);

            // Timeout after 10 seconds
            setTimeout(() => {
                if (this.pendingRequests.has(id)) {
                    this.pendingRequests.delete(id);
                    reject(new Error(`Timeout calling ${name}`));
                }
            }, 10000);
        });
    }

    private sendRaw(msg: any) {
        const json = JSON.stringify(msg);
        this.process?.stdin?.write(json + '\n');
    }

    private processBuffer() {
        const lines = this.buffer.split('\n');
        this.buffer = lines.pop() || '';

        for (const line of lines) {
            if (!line.trim()) continue;
            try {
                const msg = JSON.parse(line);
                if (msg.id !== undefined && this.pendingRequests.has(msg.id)) {
                    const pending = this.pendingRequests.get(msg.id)!;
                    this.pendingRequests.delete(msg.id);

                    if (msg.error) {
                        pending.reject(new Error(msg.error.message));
                    } else {
                        // Extract text content from MCP response
                        const result = msg.result;
                        if (result?.content?.[0]?.text) {
                            try {
                                pending.resolve(JSON.parse(result.content[0].text));
                            } catch {
                                pending.resolve(result.content[0].text);
                            }
                        } else {
                            pending.resolve(result);
                        }
                    }
                }
            } catch {
                // Ignore parse errors (stderr output mixed in)
            }
        }
    }

    private findBinary(): string | null {
        // Check common locations
        const locations = [
            'teamcontext', // In PATH
            path.join(process.env.HOME || '', 'go', 'bin', 'teamcontext'),
            '/usr/local/bin/teamcontext',
        ];

        for (const loc of locations) {
            try {
                cp.execSync(`which ${loc} 2>/dev/null || test -x "${loc}"`);
                return loc;
            } catch {
                continue;
            }
        }

        // Try which
        try {
            return cp.execSync('which teamcontext').toString().trim();
        } catch {
            return null;
        }
    }

    dispose() {
        this.process?.kill();
        this.process = null;
    }
}
