type TicketDetails = {
    assigneeName: string;
    assigneeAvatar: string;
    labels: string[];
    description: string;
    summary: string;
    ticketId: string;
    jiraIcon: string;
    versions: string;
    statusKey: string;
    issueIcon: string;
}

type TicketData = {
    key: string;
    fields: TicketDataFields;
}

type AvatarUrls = {
    '48x48': string;
}

type Assignee = {
    displayName: string;
    avatarUrls: AvatarUrls;
}

type TicketDataFields = {
    assignee: Assignee | null;
    labels: string[];
    description: string;
    summary: string;
    project: {avatarUrls: AvatarUrls};
    versions: string[];
    status: {name: string};
    issuetype: {iconUrl: string};
}

type Action = {
    type: string;
    data: TicketData;
}
