// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export function getProjectValues(metadata) {
    if (!metadata) {
        return [];
    }

    return metadata.projects;
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

    return metadata.issues_per_project[projectKey];
}

export function getIssueValuesForMultipleProjects(metadata, projectKeys) {
    return projectKeys.map((project) =>
        getIssueValues(metadata, project)).flat().filter(Boolean).sort((a, b) => a.value - b.value).filter((ele, i, me) => i === 0 || ele.value !== me[i - 1].value);
}

export function getFields(metadata, projectKey, issueTypeId) {
    if (!metadata || !projectKey || !issueTypeId) {
        return [];
    }

    return getIssueTypes(metadata, projectKey).find((issueType) => issueType.id === issueTypeId).fields;
}

export function getCustomFieldValuesForProject(metadata, projectKey) {
    if (!metadata || !projectKey) {
        return [];
    }

    const issueTypes = getIssueTypes(metadata, projectKey);

    const customFieldHash = {};
    const fields = issueTypes.map((it) => Object.values(it.fields)).flat();
    fields.forEach((field) => {
        const key = field.key;
        if (key.startsWith('customfield_')) {
            customFieldHash[key] = field;
        }
    });

    return Object.values(customFieldHash).sort((a, b) => {
        if (a.name < b.name) {
            return -1;
        }
        if (a.name > b.name) {
            return 1;
        }
        return 0;
    }).map((field) => ({
        label: `Issue Updated: Custom - ${field.name}`,
        value: `event_updated_${field.key}`,
    }));
}

export function getFieldValues(metadata, projectKey, issueTypeId) {
    const fieldsForIssue = getFields(metadata, projectKey, issueTypeId);
    const fieldIds = Object.keys(fieldsForIssue);
    return fieldIds.map((fieldId) => ({value: fieldId, label: fieldsForIssue[fieldId].name}));
}
