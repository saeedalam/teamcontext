import * as vscode from 'vscode';
import { TeamBrainClient } from '../client';

export class ExpertsProvider implements vscode.TreeDataProvider<ExpertItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<ExpertItem | undefined>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;
    private currentFile: string | undefined;

    constructor(private client: TeamBrainClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire(undefined);
    }

    async showForFile(filePath: string): Promise<void> {
        this.currentFile = filePath;
        this.refresh();
    }

    getTreeItem(element: ExpertItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<ExpertItem[]> {
        if (!this.currentFile) {
            return [new ExpertItem('Open a file and run "Show Experts"', '', vscode.TreeItemCollapsibleState.None)];
        }

        try {
            const data = await this.client.callTool('find_experts', { files: [this.currentFile] });
            if (!data?.experts || data.experts.length === 0) {
                return [new ExpertItem('No experts found', this.currentFile, vscode.TreeItemCollapsibleState.None)];
            }

            return data.experts.map((e: any) => {
                const status = e.active ? 'active' : 'INACTIVE';
                const item = new ExpertItem(
                    `${e.name} (${Math.round(e.ownership * 100)}%)`,
                    status,
                    vscode.TreeItemCollapsibleState.None
                );
                item.iconPath = new vscode.ThemeIcon(e.active ? 'person' : 'person-outline');
                item.tooltip = `${e.name}\nOwnership: ${Math.round(e.ownership * 100)}%\nStatus: ${status}\nEmail: ${e.email || 'N/A'}`;
                return item;
            });
        } catch {
            return [new ExpertItem('Unable to load experts', '', vscode.TreeItemCollapsibleState.None)];
        }
    }
}

class ExpertItem extends vscode.TreeItem {
    constructor(label: string, public description: string, collapsibleState: vscode.TreeItemCollapsibleState) {
        super(label, collapsibleState);
    }
}
