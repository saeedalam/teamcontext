import * as vscode from 'vscode';
import { TeamContextClient } from '../client';

export class DecisionsProvider implements vscode.TreeDataProvider<DecisionItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<DecisionItem | undefined>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    constructor(private client: TeamContextClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire(undefined);
    }

    getTreeItem(element: DecisionItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<DecisionItem[]> {
        try {
            const data = await this.client.callTool('list_decisions');
            if (!data?.decisions) return [];

            return data.decisions
                .filter((d: any) => d.status === 'active')
                .slice(0, 30)
                .map((d: any) => {
                    const item = new DecisionItem(
                        d.content,
                        d.reason || '',
                        vscode.TreeItemCollapsibleState.None
                    );
                    item.iconPath = new vscode.ThemeIcon('check');
                    item.tooltip = `${d.id}\n${d.content}\nReason: ${d.reason || 'N/A'}\nFeature: ${d.feature || 'global'}`;
                    return item;
                });
        } catch {
            return [];
        }
    }
}

class DecisionItem extends vscode.TreeItem {
    constructor(label: string, public description: string, collapsibleState: vscode.TreeItemCollapsibleState) {
        super(label, collapsibleState);
    }
}
