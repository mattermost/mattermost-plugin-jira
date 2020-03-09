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

export enum JiraFieldCustomTypeEnums {
    SPRINT = 'com.pyxis.greenhopper.jira:gh-sprint',
    EPIC_LINK = 'com.pyxis.greenhopper.jira:gh-epic-link',
    EPIC_NAME = 'com.pyxis.greenhopper.jira:gh-epic-label',
    RANK = 'com.pyxis.greenhopper.jira:gh-lexo-rank',
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

// Collection of ChannelSubscriptions
export type ChannelSubscriptions = {[subscriptionId: string]: ChannelSubscription};

export enum JiraInstanceType {
    SERVER = 'server',
    CLOUD = 'cloud',
}

export type CreateModal = {
    postId?: string;
    description?: string;
    channelId?: string;
}

export type PluginSettings = {
    ui_enabled: boolean;
}

export type PluginState = {
    createModalVisible: boolean;
    createModal: CreateModal;
    attachCommentToIssueModalVisible: boolean;
    attachCommentToIssueModalForPostId: string;
    jiraIssueMetadata: IssueMetadata;
    jiraProjectMetadata: ProjectMetadata;
    channelIdWithSettingsOpen: string;
    channelSubscriptions: ChannelSubscriptions;
    userConnected: boolean;
    instanceInstalled: boolean;
    instanceType: JiraInstanceType;
    pluginSettings: PluginSettings;
}

export type CreateModalValue = {
    postId: string;
    description: string;
    channelId: string;
} | {};
