// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ProjectMetadata, ReactSelectOption, IssueMetadata, IssueType, JiraField, FilterField, SelectField, StringArrayField, IssueTypeIdentifier} from 'types/model';

type FieldWithInfo = JiraField & {
    changeLogID: string;
    topLevelKey: string;
    validIssueTypes: IssueTypeIdentifier[];
    issueTypeMeta: IssueTypeIdentifier;
}

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

export function getProjectValues(metadata: ProjectMetadata): ReactSelectOption[] {
    if (!metadata || !metadata.projects) {
        return [];
    }

    return metadata.projects;
}

export function getIssueTypes(metadata: IssueMetadata, projectKey: string): IssueType[] {
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

export function getFields(metadata: IssueMetadata, projectKey: string, issueTypeId: string): {[key: string]: JiraField} {
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
];

const avoidedCustomTypes = [
    'com.pyxis.greenhopper.jira:gh-sprint',
];

const acceptedCustomTypes = [
    'com.pyxis.greenhopper.jira:gh-epic-link',
];

function isValidFieldForFilter(field: JiraField): boolean {
    const {custom, type, items} = field.schema;
    if (custom && avoidedCustomTypes.includes(custom)) {
        return false;
    }

    return allowedTypes.includes(type) || (custom && acceptedCustomTypes.includes(custom)) ||
    type === 'option' ||
    (type === 'array' && items === 'option') ||
    (type === 'array' && items === 'version') ||
    (type === 'array' && items === 'string');
}

export function getCustomFieldFiltersForProjects(metadata: IssueMetadata | null, projectKeys: string[]): FilterField[] {
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

    const result = populatedFields.concat(userDefinedFields);
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

    return sortByName(result);
}

export function getCustomFieldValuesForProjects(metadata: IssueMetadata | null, projectKeys: string[]): ReactSelectOption[] {
    return getCustomFieldsForProjects(metadata, projectKeys).map((field) => ({
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
    return field.schema && field.schema.custom === 'com.pyxis.greenhopper.jira:gh-epic-label';
}

export function isEpicLinkField(field: JiraField | FilterField): boolean {
    return field.schema && field.schema.custom === 'com.pyxis.greenhopper.jira:gh-epic-link';
}

export function isEpicIssueType(issueType: IssueType): boolean {
    return issueType.name === 'Epic';
}
