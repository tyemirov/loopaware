// @ts-check
import { spawn } from 'node:child_process';

/** Run the gateway native builder and terminate its process group on interruption.
 * @param {string} gateway
 * @param {{source_root: string}} request
 * @param {NodeJS.ProcessEnv} environment
 * @param {AbortSignal} [signal]
 * @returns {Promise<number>}
 */
export async function runNativeBuild(gateway, request, environment, signal) {
    const interruptedStatus = () => signal?.reason === 'SIGINT' ? 130 : 143;
    if (signal?.aborted) return interruptedStatus();
    return new Promise((resolve, reject) => {
        const child = spawn(gateway, ['mobile-build-operation'], {
            cwd: request.source_root, env: environment,
            detached: true, stdio: ['pipe', 'inherit', 'inherit']
        });
        const interrupt = () => {
            if (!child.pid) return;
            try { process.kill(-child.pid, signal?.reason === 'SIGINT' ? 'SIGINT' : 'SIGTERM'); }
            catch (error) { if (!(error instanceof Error && 'code' in error && error.code === 'ESRCH')) reject(error); }
        };
        signal?.addEventListener('abort', interrupt, { once: true });
        if (signal?.aborted) interrupt();
        child.once('error', () => {
            signal?.removeEventListener('abort', interrupt);
            reject(new Error('The authoritative gateway native builder could not start.'));
        });
        child.once('close', status => {
            signal?.removeEventListener('abort', interrupt);
            resolve(signal?.aborted ? interruptedStatus() : status ?? 1);
        });
        // EPIPE is followed by close; preserve the native command's exit status.
        child.stdin.on('error', () => {});
        child.stdin.end(JSON.stringify(request));
    });
}
