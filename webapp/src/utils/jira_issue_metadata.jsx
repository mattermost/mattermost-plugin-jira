// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export function getProjectValues(metadata) {
    if (!metadata) {
        return [];
    }

    return metadata.projects.map((p) => ({value: p.key, label: p.name}));
}

export function getIssueTypes(metadata, projectKey) {
    const project = metadata.projects.find((proj) => proj.key === projectKey);
    if (!project) {
        return [];
    }
    return project.issuetypes.filter((i) => !i.subtask);
}

export function getIssueValues(metadata, projectKey) {
    if (!metadata || !projectKey) {
        return [];
    }

    return getIssueTypes(metadata, projectKey).map((issueType) => ({value: issueType.id, label: issueType.name}));
}

export function getIssueValuesForMultipleProjects(metadata, projectKeys) {
    return projectKeys.map((project) =>
        getIssueValues(metadata, project)).
        flat().
        sort((a, b) => a.value - b.value).
        filter((ele, i, me) => i === 0 || ele.value !== me[i - 1].value);
}

export function getIssueTypesForMultipleProjects(metadata, projectKeys) {
    return projectKeys.map((project) =>
        getIssueTypes(metadata, project)).
        flat().
        sort((a, b) => a.id - b.id).
        filter((ele, i, me) => i === 0 || ele.id !== me[i - 1].id);
}

export function getFieldsForMultipleProjects(metadata, projectKeys) {
    if (!metadata || !projectKeys) {
        return [];
    }

    return getIssueTypesForMultipleProjects(metadata, projectKeys).map((issue) => issue.fields).reduce((acc, cur) => Object.assign(acc, cur), {});
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

