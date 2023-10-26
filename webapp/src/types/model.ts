export type ReactSelectOption = {
    label: string | React.ReactElement;
    value: string;
};

export type AllowedValue = {
    id: string;
    value?: string;
    name?: string;
};

export type FieldSchema = {
    type: string;
    system?: string;
    custom?: JiraFieldCustomTypeEnums;
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

export type IssueTypeWithStatuses = {
    id: string;
    name: string;
    statuses: Status[];
}

export type IssueMetadata = {
    projects: Project[];
    issue_types: IssueTypeWithStatuses[];
}

export type Status = {
    id: string;
    name: string;
}

export type ProjectMetadata = {
    projects: ReactSelectOption[];
    issues_per_project: {[key: string]: ReactSelectOption[]};
    default_project_key?: string;
}

export enum JiraFieldTypeEnums {
    PROJECT = 'project',
    ISSUE_TYPE = 'issuetype',
    PRIORITY = 'priority',
    DESCRIPTION = 'description',
    SUMMARY = 'summary',
    LABELS = 'labels',
    ASSIGNEE = 'assignee',
    SECURITY = 'security',
    COMPONENTS = 'components',
    FIX_VERSIONS = 'fixVersions',
    AFFECTS_VERSIONS = 'versions',
}

export enum AvatarSize {
    XSMALL = '16x16',
    SMALL = '24x24',
    MEDIUM = '36x36',
    LARGE = '48x48',
}

export type JiraUser = {
    self: string;
    accountId: string;
    avatarUrls: {[size in AvatarSize]: string};
    displayName: string;
    active: boolean;
    timeZone: string;
}

export enum JiraFieldCustomTypeEnums {
    SPRINT = 'com.pyxis.greenhopper.jira:gh-sprint',
    EPIC_LINK = 'com.pyxis.greenhopper.jira:gh-epic-link',
    EPIC_NAME = 'com.pyxis.greenhopper.jira:gh-epic-label',
    RANK = 'com.pyxis.greenhopper.jira:gh-lexo-rank',

    CASCADING_SELECT = 'com.atlassian.jira.plugin.system.customfieldtypes:cascadingselect',
    TEXT_AREA = 'com.atlassian.jira.plugin.system.customfieldtypes:textarea',
    TEXT_FIELD = 'com.atlassian.jira.plugin.system.customfieldtypes:textfield',
    SELECT = 'com.atlassian.jira.plugin.system.customfieldtypes:select',
    PROJECT = 'com.atlassian.jira.plugin.system.customfieldtypes:project',

    MULTI_SELECT = 'com.atlassian.jira.plugin.system.customfieldtypes:multiselect',
    RADIO_BUTTONS = 'com.atlassian.jira.plugin.system.customfieldtypes:radiobuttons',
    MULTI_CHECKBOXES = 'com.atlassian.jira.plugin.system.customfieldtypes:multicheckboxes',
    URL_FIELD = 'com.atlassian.jira.plugin.system.customfieldtypes:url',

    LABELS = 'com.atlassian.jira.plugin.system.customfieldtypes:labels',
    USER_PICKER = 'com.atlassian.jira.plugin.system.customfieldtypes:userpicker',
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
    instance_id: string;
}

export enum InstanceType {
    CLOUD = 'cloud',
    SERVER = 'server',
}
export type Instance = {
    alias?: string;
    instance_id: string;
    type: InstanceType;
}

export type GetConnectedResponse = {
    data: {
        can_connect: boolean;
        instances: Instance[];
        is_connected: boolean;
        user: {
            connected_instances: Instance[];
            default_instance_id?: string;
        };
    };
    error?: Error;
};

export type APIResponse<T> = {
    error?: Error;
    data: T;
};

export type CreateIssueRequest = {
    instance_id: string;
    required_fields_not_covered: string[][];
    post_id: string;
    current_team: string;
    channel_id: string;
    fields: {};
};

export type SearchIssueParams = {
    jql?: string;
    fields: string;
    q: string;
    instance_id: string;
};

export type AttachCommentRequest = {
    post_id: string;
    current_team: string;
    issueKey: string;
    instance_id: string;
};

export type AllProjectMetadata = {
    instance_id: string;
    metadata: ProjectMetadata;
}[];

export type CreateIssueFields = {
    description: string;
    summary?: string;
    project: {key: string};
    issuetype: {id: string};
} & {[key: string]: JiraField};
