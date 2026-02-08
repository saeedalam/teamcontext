import * as vscode from 'vscode';
import { TeamBrainClient } from '../client';

export class WarningsProvider implements vscode.TreeDataProvider<WarningItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<WarningItem | undefined>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    constructor(private client: TeamBrainClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire(undefined);
    }

    getTreeItem(element: WarningItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<WarningItem[]> {
        try {
            const data = await this.client.callTool('list_warnings');
            if (!data?.warnings) return [];

            return data.warnings.slice(0, 30).map((w: any) => {
                const item = new WarningItem(
                    w.content,
                    w.severity || 'warning',
                    vscode.TreeItemCollapsibleState.None
                );
                item.iconPath = new vscode.ThemeIcon(
                    w.severity === 'critical' ? 'error' : 'warning'
                );
                item.tooltip = `${w.id}\n${w.content}\nSeverity: ${w.severity}\nReason: ${w.reason || 'N/A'}`;
                return item;
            });
        } catch {
            return [];
        }
    }
}

class WarningItem extends vscode.TreeItem {
    constructor(label: string, public description: string, collapsibleState: vscode.TreeItemCollapsibleState) {
        super(label, collapsibleState);
    }
}
