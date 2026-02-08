import * as vscode from 'vscode';
import { TeamContextClient } from '../client';

export class FeedProvider implements vscode.TreeDataProvider<FeedItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<FeedItem | undefined>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    constructor(private client: TeamContextClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire(undefined);
    }

    getTreeItem(element: FeedItem): vscode.TreeItem {
        return element;
    }

    async getChildren(): Promise<FeedItem[]> {
        try {
            const data = await this.client.callTool('get_feed', { limit: 30 });
            if (!data?.entries || data.entries.length === 0) {
                return [new FeedItem('No recent activity', '', vscode.TreeItemCollapsibleState.None)];
            }

            return data.entries.map((e: any) => {
                const icon = this.typeIcon(e.type);
                const age = this.formatAge(e.created_at);
                const item = new FeedItem(
                    e.title,
                    `${e.type} - ${age}`,
                    vscode.TreeItemCollapsibleState.None
                );
                item.iconPath = new vscode.ThemeIcon(icon);
                item.tooltip = `${e.type}: ${e.title}\n${e.detail || ''}\n${age}`;
                return item;
            });
        } catch {
            return [new FeedItem('Unable to load feed', 'Check teamcontext is running', vscode.TreeItemCollapsibleState.None)];
        }
    }

    private typeIcon(type: string): string {
        switch (type) {
            case 'decision': return 'check';
            case 'warning': return 'warning';
            case 'pattern': return 'symbol-structure';
            case 'insight': return 'lightbulb';
            case 'conversation': return 'comment-discussion';
            case 'event': return 'history';
            default: return 'circle-outline';
        }
    }

    private formatAge(dateStr: string): string {
        const d = new Date(dateStr);
        const now = new Date();
        const diffMs = now.getTime() - d.getTime();
        const diffH = Math.floor(diffMs / (1000 * 60 * 60));
        if (diffH < 1) return 'just now';
        if (diffH < 24) return `${diffH}h ago`;
        const diffD = Math.floor(diffH / 24);
        if (diffD === 1) return 'yesterday';
        return `${diffD}d ago`;
    }
}

class FeedItem extends vscode.TreeItem {
    constructor(label: string, public description: string, collapsibleState: vscode.TreeItemCollapsibleState) {
        super(label, collapsibleState);
    }
}
