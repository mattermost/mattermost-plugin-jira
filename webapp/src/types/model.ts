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
    schema: FieldSchema;
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

export type JiraIssue = {
    id: string;
    key: string;
    name: string;
    fields: {[key: string]: JiraField};
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

export enum FilterFieldInclusion {
    INCLUDE_ANY = 'include_any',
    INCLUDE_ALL = 'include_all',
    EXCLUDE_ANY = 'exclude_any',
    EMPTY = 'empty',
}

export type FilterValue = {
    key: string;
    values: string[];
    inclusion: FilterFieldInclusion;
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
    name: string;
}
