type TicketDetails = {
    assigneeName: string;
    assigneeAvatar: any;
    labels: any;
    description: string;
    summary: any;
    ticketId: any;
    jiraIcon: any;
    versions: any;
    statusKey: any;
    issueIcon: any;
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
