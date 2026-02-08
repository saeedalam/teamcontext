import * as vscode from 'vscode';
import { RisksProvider } from './providers/risks';
import { DecisionsProvider } from './providers/decisions';
import { WarningsProvider } from './providers/warnings';
import { ExpertsProvider } from './providers/experts';
import { FeedProvider } from './providers/feed';
import { TeamContextClient } from './client';
import { ComplianceChecker } from './compliance';

let client: TeamContextClient;

export function activate(context: vscode.ExtensionContext) {
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) {
        return;
    }

    client = new TeamContextClient(workspaceRoot);

    // Register tree data providers
    const risksProvider = new RisksProvider(client);
    const decisionsProvider = new DecisionsProvider(client);
    const warningsProvider = new WarningsProvider(client);
    const expertsProvider = new ExpertsProvider(client);
    const feedProvider = new FeedProvider(client);

    vscode.window.registerTreeDataProvider('teamcontext.risks', risksProvider);
    vscode.window.registerTreeDataProvider('teamcontext.decisions', decisionsProvider);
    vscode.window.registerTreeDataProvider('teamcontext.warnings', warningsProvider);
    vscode.window.registerTreeDataProvider('teamcontext.experts', expertsProvider);
    vscode.window.registerTreeDataProvider('teamcontext.feed', feedProvider);

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('teamcontext.refresh', () => {
            risksProvider.refresh();
            decisionsProvider.refresh();
            warningsProvider.refresh();
            expertsProvider.refresh();
            feedProvider.refresh();
        }),
        vscode.commands.registerCommand('teamcontext.checkCompliance', () => {
            const editor = vscode.window.activeTextEditor;
            if (editor) {
                new ComplianceChecker(client).checkFile(editor.document.uri.fsPath);
            }
        }),
        vscode.commands.registerCommand('teamcontext.showExperts', () => {
            const editor = vscode.window.activeTextEditor;
            if (editor) {
                expertsProvider.showForFile(editor.document.uri.fsPath);
            }
        }),
        vscode.commands.registerCommand('teamcontext.sync', async () => {
            const terminal = vscode.window.createTerminal('TeamContext Sync');
            terminal.sendText('teamcontext sync');
            terminal.show();
        }),
        vscode.commands.registerCommand('teamcontext.feed', () => {
            feedProvider.refresh();
        })
    );

    // Real-time compliance on save
    context.subscriptions.push(
        vscode.workspace.onDidSaveTextDocument((doc) => {
            new ComplianceChecker(client).checkFile(doc.uri.fsPath);
        })
    );

    vscode.window.showInformationMessage('TeamContext activated');
}

export function deactivate() {
    if (client) {
        client.dispose();
    }
}
