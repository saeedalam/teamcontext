import * as vscode from 'vscode';
import { TeamContextClient } from '../client';

export class RisksProvider implements vscode.TreeDataProvider<RiskItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<RiskItem | undefined>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    constructor(private client: TeamContextClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire(undefined);
    }

    getTreeItem(element: RiskItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<RiskItem[]> {
        try {
            const data = await this.client.callTool('get_knowledge_risks');
            if (!data?.risks) return [];

            return data.risks.map((r: any) => {
                const item = new RiskItem(
                    `${r.level}: ${r.area}`,
                    r.description || r.reason || '',
                    vscode.TreeItemCollapsibleState.None
                );
                item.iconPath = new vscode.ThemeIcon(
                    r.level === 'CRITICAL' ? 'error' : r.level === 'HIGH' ? 'warning' : 'info'
                );
                item.tooltip = `${r.level}\n${r.area}\n${r.description || ''}`;
                return item;
            });
        } catch {
            return [new RiskItem('Unable to load risks', 'Check teamcontext is running', vscode.TreeItemCollapsibleState.None)];
        }
    }
}

class RiskItem extends vscode.TreeItem {
    constructor(label: string, public description: string, collapsibleState: vscode.TreeItemCollapsibleState) {
        super(label, collapsibleState);
    }
}
