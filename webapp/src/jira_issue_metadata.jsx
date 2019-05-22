// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export function getProjectValues(metadata) {
    if (!metadata) {
        return [];
    }

    return metadata.projects.map((p) => ({value: p.key, label: p.name}));
}

export function getIssueTypes(metadata, projectKey) {
    return metadata.projects.find((project) => project.key === projectKey).issuetypes.filter((i) => !i.subtask);
}

export function getIssueValues(metadata, projectKey) {
    if (!metadata || !projectKey) {
        return [];
    }

    return getIssueTypes(metadata, projectKey).map((issueType) => ({value: issueType.id, label: issueType.name}));
}

export function getFields(metadata, projectKey, issueTypeId) {
    if (!metadata || !projectKey || !issueTypeId) {
        return [];
    }

    return getIssueTypes(metadata, projectKey).find((issueType) => issueType.id === issueTypeId).fields;
}

export function getFieldValues(metadata, projectKey, issueTypeId) {
    const fieldsForIssue = getFields(metadata, projectKey, issueTypeId);
    const fieldIds = Object.keys(fieldsForIssue);
    return fieldIds.map((fieldId) => ({value: fieldId, label: fieldsForIssue[fieldId].name}));
}

