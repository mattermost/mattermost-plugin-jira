// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {
    ProjectMetadata,
    ReactSelectOption,
    IssueMetadata,
    IssueType,
    JiraField,
    FilterField,
    FilterValue,
    SelectField,
    StringArrayField,
    IssueTypeIdentifier,
    ChannelSubscriptionFilters,
    FilterFieldInclusion,
    JiraFieldCustomTypeEnums,
    JiraFieldTypeEnums,
    Status,
} from 'types/model';
import {TicketData, TicketDetails} from 'types/tooltip';

type FieldWithInfo = JiraField & {
    changeLogID: string;
    topLevelKey: string;
    validIssueTypes: IssueTypeIdentifier[];
    issueTypeMeta: IssueTypeIdentifier;
}

const commentVisibilityFieldKey = 'commentVisibility';

export const FIELD_KEY_STATUS = 'status';

// This is a replacement for the Array.flat() function which will be polyfilled by Babel
// in our 5.16 release. Remove this and replace with .flat() then.
const flatten = (arr: any[]) => {
    return arr.reduce((acc, val) => acc.concat(val), []);
};

function sortByName<T>(arr: (T & {name: string})[]): T[] {
    return arr.sort((a, b) => {
        if (a.name < b.name) {
            return -1;
        }
        if (a.name > b.name) {
            return 1;
        }
        return 0;
    });
}

export function getProjectValues(metadata: ProjectMetadata | null): ReactSelectOption[] {
    if (!metadata || !metadata.projects) {
        return [];
    }

    return metadata.projects;
}

export function getIssueTypes(metadata: IssueMetadata | null, projectKey: string | null): IssueType[] {
    if (!metadata || !metadata.projects) {
        return [];
    }

    const project = metadata.projects.find((proj) => proj.key === projectKey);
    if (!project) {
        return [];
    }
    return project.issuetypes.filter((i) => !i.subtask);
}

export function getIssueValues(metadata: ProjectMetadata, projectKey: string): ReactSelectOption[] {
    if (!metadata || !metadata.issues_per_project || !projectKey) {
        return [];
    }

    return metadata.issues_per_project[projectKey];
}

export function getIssueValuesForMultipleProjects(metadata: ProjectMetadata, projectKeys: string[]): ReactSelectOption[] {
    if (!metadata || !metadata.projects || !projectKeys) {
        return [];
    }

    const issueValues = flatten(projectKeys.map((project) => getIssueValues(metadata, project))).filter(Boolean);

    const issueTypeHash: {[key: string]: ReactSelectOption} = {};
    issueValues.forEach((issueType: ReactSelectOption) => {
        issueTypeHash[issueType.value] = issueType;
    });

    return Object.values(issueTypeHash);
}

export function getFields(metadata: IssueMetadata | null, projectKey: string | null, issueTypeId: string | null): {[key: string]: JiraField} {
    if (!metadata || !projectKey || !issueTypeId) {
        return {};
    }

    const issueType = getIssueTypes(metadata, projectKey).find((it) => it.id === issueTypeId);
    if (issueType) {
        return issueType.fields;
    }
    return {};
}

export function getConflictingFields(fields: FilterField[], chosenIssueTypes: string[], issueMetadata: IssueMetadata): {field: FilterField; issueTypes: IssueType[]}[] {
    const conflictingFields = [];

    for (const field of fields) {
        const conflictingIssueTypes = [];
        for (const issueTypeId of chosenIssueTypes) {
            const issueTypes = field.issueTypes;
            if (!issueTypes.find((it) => it.id === issueTypeId)) {
                const issueType = issueMetadata.projects[0].issuetypes.find((i) => i.id === issueTypeId) as IssueType;
                conflictingIssueTypes.push(issueType);
            }
        }
        if (conflictingIssueTypes.length) {
            conflictingFields.push({field, issueTypes: conflictingIssueTypes});
        }
    }
    return conflictingFields;
}

export function getCustomFieldsForProjects(metadata: IssueMetadata | null, projectKeys: string[]): FieldWithInfo[] {
    if (!metadata || !projectKeys || !projectKeys.length) {
        return [];
    }

    const issueTypes = flatten(projectKeys.map((key) => getIssueTypes(metadata, key))) as IssueType[];

    const customFieldHash: {[key: string]: FieldWithInfo} = {};
    const fields = flatten(issueTypes.map((issueType) =>
        Object.keys(issueType.fields).map((key) => ({
            ...issueType.fields[key],
            topLevelKey: key,
            issueTypeMeta: {
                id: issueType.id,
                name: issueType.name,
            },
        }))
    )).filter(Boolean) as FieldWithInfo[];

    for (const field of fields) {
        // Jira server webhook fields don't have keys
        // name is the most unique property available in that case
        const changeLogID = field.key || field.name;
        let current = customFieldHash[field.topLevelKey];
        if (!current) {
            current = {...field, changeLogID, key: field.key || field.topLevelKey, validIssueTypes: []};
        }
        current.validIssueTypes.push(field.issueTypeMeta);

        customFieldHash[field.topLevelKey] = current;
    }

    return sortByName(Object.values(customFieldHash));
}

const allowedTypes = [
    'priority',
    'securitylevel',
    'security',
];

const allowedArrayTypes = [
    'component',
    'option', // multiselect
    'string', // labels
    'version', // fix and affects versions
];

const avoidedCustomTypesForFilters: string[] = [
    JiraFieldCustomTypeEnums.SPRINT,
];

const acceptedCustomTypesForFilters: string[] = [
    JiraFieldCustomTypeEnums.EPIC_LINK,
    JiraFieldCustomTypeEnums.CASCADING_SELECT,
];

function isValidFieldForFilter(field: JiraField): boolean {
    const {custom, type, items} = field.schema;
    if (custom && avoidedCustomTypesForFilters.includes(custom)) {
        return false;
    }

    return allowedTypes.includes(type) || (custom && acceptedCustomTypesForFilters.includes(custom)) ||
    type === 'option' || // single select
    (type === 'array' && allowedArrayTypes.includes(items));
}

export function getStatusField(metadata: IssueMetadata | null, selectedIssueTypes: string[]): FilterField | null {
    // Filtering out the statuses on the basis of selected issue types
    const issueTypesWithStatuses = metadata && metadata.issue_types_with_statuses;
    const keys = new Set<string>();
    const statuses: Status[] = [];
    if (issueTypesWithStatuses) {
        for (const issueType of issueTypesWithStatuses) {
            if (selectedIssueTypes.includes(issueType.id) || !selectedIssueTypes.length) {
                for (const status of issueType.statuses) {
                    if (!keys.has(status.id)) {
                        keys.add(status.id);
                        statuses.push(status);
                    }
                }
            }
        }
    }

    if (!statuses.length) {
        return null;
    }

    return {
        key: FIELD_KEY_STATUS,
        name: 'Status',
        schema: {
            type: 'array',
        },
        values: statuses.map((value) => ({
            label: value.name,
            value: value.id,
        })),
        issueTypes: metadata && metadata.issue_types_with_statuses.map((type) => {
            return {
                id: type.id,
                name: type.name,
            };
        }),
    } as FilterField;
}

export function getCustomFieldFiltersForProjects(metadata: IssueMetadata | null, projectKeys: string[], issueTypes: string[]): FilterField[] {
    const fields = getCustomFieldsForProjects(metadata, projectKeys).filter(isValidFieldForFilter);
    const selectFields = fields.filter((field) => Boolean(field.allowedValues && field.allowedValues.length)) as (SelectField & FieldWithInfo)[];
    const populatedFields = selectFields.map((field) => {
        return {
            key: field.key,
            name: field.name,
            schema: field.schema,
            values: field.allowedValues.map((value) => ({
                label: value.name || value.value,
                value: value.id,
            })),
            issueTypes: field.validIssueTypes,
        } as FilterField;
    });

    const stringArrayFields = fields.filter((field) => field.schema.type === 'array' && field.schema.items === 'string' && !field.allowedValues) as (StringArrayField & FieldWithInfo)[];
    const userDefinedFields = stringArrayFields.map((field) => {
        return {
            key: field.key,
            name: field.name,
            schema: field.schema,
            userDefined: true,
            issueTypes: field.validIssueTypes,
        } as FilterField;
    });

    const commentVisibility = selectFields.map((field) => {
        return {
            key: commentVisibilityFieldKey,
            name: 'Comment Visibility',
            schema: {
                type: 'commentVisibility',
            },
            values: [],
            issueTypes: field.validIssueTypes,
        } as FilterField;
    });

    const result = populatedFields.concat(userDefinedFields, commentVisibility);
    const epicLinkField = fields.find(isEpicLinkField);
    if (epicLinkField) {
        result.unshift({
            key: epicLinkField.key,
            name: epicLinkField.name,
            schema: epicLinkField.schema,
            values: [],
            issueTypes: epicLinkField.validIssueTypes,
        } as FilterField);
    }

    const statusField = getStatusField(metadata, issueTypes);
    if (statusField) {
        result.push(statusField);
    }

    return sortByName(result);
}

const avoidedCustomTypesForEvents: string[] = [
    JiraFieldCustomTypeEnums.SPRINT,
    JiraFieldCustomTypeEnums.RANK,
];

function isValidFieldForEvents(field: JiraField): boolean {
    const {custom} = field.schema;
    if (!custom) {
        return false;
    }

    return !avoidedCustomTypesForEvents.includes(custom);
}

export function getCustomFieldValuesForEvents(metadata: IssueMetadata | null, projectKeys: string[]): ReactSelectOption[] {
    return getCustomFieldsForProjects(metadata, projectKeys).filter(isValidFieldForEvents).map((field) => ({
        label: `Issue Updated: Custom - ${field.name}`,
        value: `event_updated_${field.changeLogID}`,
    }));
}

export function getFieldValues(metadata: IssueMetadata, projectKey: string, issueTypeId: string): ReactSelectOption[] {
    const fieldsForIssue = getFields(metadata, projectKey, issueTypeId);
    const fieldIds = Object.keys(fieldsForIssue);
    return fieldIds.map((fieldId) => ({value: fieldId, label: fieldsForIssue[fieldId].name}));
}

export function isEpicNameField(field: JiraField | FilterField): boolean {
    return field.schema && field.schema.custom === JiraFieldCustomTypeEnums.EPIC_NAME;
}

export function isEpicLinkField(field: JiraField | FilterField): boolean {
    return field.schema && field.schema.custom === JiraFieldCustomTypeEnums.EPIC_LINK;
}

export function isLabelField(field: JiraField | FilterField): boolean {
    return field.schema.system === 'labels' || field.schema.custom === 'com.atlassian.jira.plugin.system.customfieldtypes:labels';
}

export function isCommentVisibilityField(field: JiraField | FilterField): boolean {
    return field.key === commentVisibilityFieldKey;
}

export function isEpicIssueType(issueType: IssueType): boolean {
    return issueType.name === 'Epic';
}

export function isMultiSelectField(field: FilterField): boolean {
    return field.schema.type === 'array';
}

export function isTextField(field: JiraField | FilterField): boolean {
    return field.schema.type === 'string';
}

export function isSecurityLevelField(field: JiraField | FilterField): boolean {
    return field.schema.type === 'securitylevel';
}

export function filterValueIsSecurityField(value: FilterValue): boolean {
    return value.key === JiraFieldTypeEnums.SECURITY;
}

// Some Jira fields have special names for JQL
function getFieldNameForJQL(field: FilterField) {
    switch (field.key) {
    case 'fixVersions':
        return 'fixVersion';
    case 'versions':
        return 'affectedVersion';
    default:
        return field.name;
    }
}

function quoteGuard(s: string) {
    if (s && s.includes(' ')) {
        return `"${s}"`;
    }

    return s;
}

export function generateJQLStringFromSubscriptionFilters(issueMetadata: IssueMetadata, fields: FilterField[], filters: ChannelSubscriptionFilters, securityLevelEmptyForJiraSubscriptions: boolean) {
    const projectJQL = `Project = ${quoteGuard(filters.projects[0]) || '?'}`;

    let issueTypeValueString = '?';
    if (filters.issue_types.length) {
        const issueTypeNames = filters.issue_types.map((issueTypeId) => {
            const issueType = issueMetadata.projects[0].issuetypes.find((it) => it.id === issueTypeId);
            if (!issueType) {
                return issueTypeId;
            }

            return `${quoteGuard(issueType.name)}`;
        });
        issueTypeValueString = `(${issueTypeNames.join(', ')})`;
    }
    const issueTypesJQL = `IssueType IN ${issueTypeValueString}`;

    let filterFieldsJQL = filters.fields.map(({key, inclusion, values}): string => {
        const field = fields.find((f) => f.key === key);
        if (!field) {
            // broken filter
            return `(cannot find field ${key})`;
        }

        const fieldName = getFieldNameForJQL(field);

        if (inclusion === FilterFieldInclusion.EMPTY) {
            return `${quoteGuard(fieldName)} IS EMPTY`;
        }

        const inclusionString = inclusion === FilterFieldInclusion.EXCLUDE_ANY ? 'NOT IN' : 'IN';
        if (!values.length) {
            return `${quoteGuard(fieldName)} ${inclusionString} ?`;
        }

        const chosenValueLabels = values.map((value) => {
            if (!(field.values && field.values.length)) {
                return value;
            }

            const found = field.values.find((v) => v.value === value);
            if (!found) {
                return value;
            }

            return found.label;
        });

        if (inclusion === FilterFieldInclusion.INCLUDE_ALL && values.length > 1) {
            const clauses = chosenValueLabels.map((v) => `${quoteGuard(fieldName)} IN (${quoteGuard(v)})`);
            return `(${clauses.join(' AND ')})`;
        }

        const joinedValues = chosenValueLabels.map((v) => `${quoteGuard(v)}`).join(', ');
        const valueString = `(${joinedValues})`;
        return `${quoteGuard(fieldName)} ${inclusionString} ${valueString}`;
    }).join(' AND ');

    const shouldShowEmptySecurityLevel = securityLevelEmptyForJiraSubscriptions && !filters.fields.some(filterValueIsSecurityField);
    if (shouldShowEmptySecurityLevel) {
        if (filterFieldsJQL.length) {
            filterFieldsJQL += ' AND ';
        }
        filterFieldsJQL += '"Security Level" IS EMPTY';
    }

    return [projectJQL, issueTypesJQL, filterFieldsJQL].filter(Boolean).join(' AND ');
}

export function getJiraTicketDetails(data?: TicketData): TicketDetails | null {
    if (!data) {
        return null;
    }

    const assignee = data && data.fields && data.fields.assignee ? data.fields.assignee : null;
    const ticketDetails: TicketDetails = {

        // TODO: Add optional chaining operator
        assigneeName: (assignee && assignee.displayName) || '',
        assigneeAvatar: (assignee && assignee.avatarUrls && assignee.avatarUrls['48x48']) || '',
        labels: data.fields && data.fields.labels,
        description: data.fields && data.fields.description,
        summary: data.fields && data.fields.summary,
        ticketId: data.key,
        jiraIcon: data.fields && data.fields.project && data.fields.project.avatarUrls && data.fields.project.avatarUrls['48x48'],
        versions: data.fields && data.fields.versions && data.fields.versions.length ? data.fields.versions[0] : '',
        statusKey: data.fields && data.fields.status && data.fields.status.name,
        issueIcon: data.fields && data.fields.issuetype && data.fields.issuetype.iconUrl,
    };
    return ticketDetails;
}
