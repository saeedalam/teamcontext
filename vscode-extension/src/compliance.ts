import * as vscode from 'vscode';
import { TeamContextClient } from './client';

const diagnosticCollection = vscode.languages.createDiagnosticCollection('teamcontext');

export class ComplianceChecker {
    constructor(private client: TeamContextClient) {}

    async checkFile(filePath: string): Promise<void> {
        try {
            const result = await this.client.callTool('check_compliance', { file_path: filePath });
            if (!result) return;

            const uri = vscode.Uri.file(filePath);
            const diagnostics: vscode.Diagnostic[] = [];

            if (result.violations) {
                for (const v of result.violations) {
                    const severity = v.severity === 'blocker'
                        ? vscode.DiagnosticSeverity.Error
                        : v.severity === 'warning'
                            ? vscode.DiagnosticSeverity.Warning
                            : vscode.DiagnosticSeverity.Information;

                    const line = v.line ? v.line - 1 : 0;
                    const range = new vscode.Range(line, 0, line, 200);
                    const diagnostic = new vscode.Diagnostic(
                        range,
                        `[TeamContext] ${v.message} (ref: ${v.reference || 'N/A'})`,
                        severity
                    );
                    diagnostic.source = 'TeamContext';
                    diagnostics.push(diagnostic);
                }
            }

            diagnosticCollection.set(uri, diagnostics);

            if (diagnostics.length === 0 && result.compliant) {
                // Clear old diagnostics
                diagnosticCollection.delete(uri);
            }
        } catch {
            // Silently fail — don't block the user
        }
    }
}
