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
    const issueValues = projectKeys.map((project) => getIssueValues(metadata, project)).flat().filter(Boolean);

    const issueTypeHash = {};
    issueValues.forEach((issueType) => {
        issueTypeHash[issueType.value] = issueType;
    });

    return Object.values(issueTypeHash);
}

export function getFields(metadata, projectKeys, issueTypeIds) {
    if (!metadata || !projectKeys || !issueTypeIds) {
        return [];
    }

    if ((Array.isArray(projectKeys) && !projectKeys.length) || (Array.isArray(issueTypeIds) && !issueTypeIds.length)) {
        return [];
    }

    const issueTypes = projectKeys.map((key) => getIssueTypes(metadata, key)).flat().filter((issueType) => issueTypeIds.includes(issueType.id));

    const fields = issueTypes.map((issueType) =>
        Object.keys(issueType.fields).map((key) => ({...issueType.fields[key], key}))
    ).flat().filter(Boolean);

    const fieldHash = {};
    for (const field of fields) {
        fieldHash[field.key] = field;
    }

    return fieldHash;
}

export function getCustomFieldValuesForProjects(metadata, projectKeys) {
    if (!metadata || !projectKeys || !projectKeys.length) {
        return [];
    }

    const issueTypes = projectKeys.map((key) => getIssueTypes(metadata, key)).flat();

    const customFieldHash = {};
    const fields = issueTypes.map((issueType) =>
        Object.keys(issueType.fields).map((key) => ({...issueType.fields[key], key}))
    ).flat().filter(Boolean);

    for (const field of fields) {
        if (field.schema.custom) {
            customFieldHash[field.key] = field;
        }
    }

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
