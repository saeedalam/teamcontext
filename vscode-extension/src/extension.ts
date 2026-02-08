import * as vscode from 'vscode';
import { RisksProvider } from './providers/risks';
import { DecisionsProvider } from './providers/decisions';
import { WarningsProvider } from './providers/warnings';
import { ExpertsProvider } from './providers/experts';
import { FeedProvider } from './providers/feed';
import { TeamBrainClient } from './client';
import { ComplianceChecker } from './compliance';

let client: TeamBrainClient;

export function activate(context: vscode.ExtensionContext) {
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) {
        return;
    }

    client = new TeamBrainClient(workspaceRoot);

    // Register tree data providers
    const risksProvider = new RisksProvider(client);
    const decisionsProvider = new DecisionsProvider(client);
    const warningsProvider = new WarningsProvider(client);
    const expertsProvider = new ExpertsProvider(client);
    const feedProvider = new FeedProvider(client);

    vscode.window.registerTreeDataProvider('teambrain.risks', risksProvider);
    vscode.window.registerTreeDataProvider('teambrain.decisions', decisionsProvider);
    vscode.window.registerTreeDataProvider('teambrain.warnings', warningsProvider);
    vscode.window.registerTreeDataProvider('teambrain.experts', expertsProvider);
    vscode.window.registerTreeDataProvider('teambrain.feed', feedProvider);

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('teambrain.refresh', () => {
            risksProvider.refresh();
            decisionsProvider.refresh();
            warningsProvider.refresh();
            expertsProvider.refresh();
            feedProvider.refresh();
        }),
        vscode.commands.registerCommand('teambrain.checkCompliance', () => {
            const editor = vscode.window.activeTextEditor;
            if (editor) {
                new ComplianceChecker(client).checkFile(editor.document.uri.fsPath);
            }
        }),
        vscode.commands.registerCommand('teambrain.showExperts', () => {
            const editor = vscode.window.activeTextEditor;
            if (editor) {
                expertsProvider.showForFile(editor.document.uri.fsPath);
            }
        }),
        vscode.commands.registerCommand('teambrain.sync', async () => {
            const terminal = vscode.window.createTerminal('TeamBrain Sync');
            terminal.sendText('teambrain sync');
            terminal.show();
        }),
        vscode.commands.registerCommand('teambrain.feed', () => {
            feedProvider.refresh();
        })
    );

    // Real-time compliance on save
    context.subscriptions.push(
        vscode.workspace.onDidSaveTextDocument((doc) => {
            new ComplianceChecker(client).checkFile(doc.uri.fsPath);
        })
    );

    vscode.window.showInformationMessage('TeamBrain activated');
}

export function deactivate() {
    if (client) {
        client.dispose();
    }
}
