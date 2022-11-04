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

type TicketDataFields = {
    labels: string[];
    description: string;
    summary: string;
    project: any;
    versions: string[];
    status: {
        name: string;
    };
    issuetype: {
        iconUrl: string;
    };
}
