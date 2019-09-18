export type ReactSelectOption = {
    label: string;
    value: string;
};

export type AllowedValue = {
    id: string;
    value?: string;
    name?: string;
};

export type FieldSchema = {
    type: string;
    custom?: string;
    customId?: number;
    items?: string;
};

export type BaseField = {
    key?: string;
    name: string;
    required: boolean;
    schema: FieldSchema;
    allowedValues?: AllowedValue[];
};

export type SelectField = BaseField & {
    allowedValues: AllowedValue[];
};

export type StringArrayField = BaseField;

export type StringField = BaseField;

export type JiraField = SelectField | StringArrayField | StringField;

export type IssueTypeIdentifier = {id: string; name: string};

export type FilterField = {
    key: string;
    name: string;
    values?: ReactSelectOption[];
    userDefined?: boolean;
    issueTypes: IssueTypeIdentifier[];
};

export type IssueType = {
    id: string;
    name: string;
    fields: {[key: string]: JiraField};
    subtask: boolean;
}

export type Project = {
    key: string;
    issuetypes: IssueType[];
}

export type IssueMetadata = {
    projects: Project[];
}

export type ProjectMetadata = {
    projects: ReactSelectOption[];
    issues_per_project: {[key: string]: ReactSelectOption[]};
}

export type FilterValue = {
    key: string;
    values: string[];
    exclude: boolean;
}

export type ChannelSubscriptionFilters = {
    projects: string[];
    events: string[];
    issue_types: string[];
    fields: FilterValue[];
};

export type ChannelSubscription = {
    id: string;
    channel_id: string;
    filters: ChannelSubscriptionFilters;
}
