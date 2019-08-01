// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// This is a replacement for the Array.flat() function which will be polyfilled by Babel
// in our 5.16 release. Remove this and replace with .flat() then.
const flatten = (arr) => {
    return arr.reduce((acc, val) => acc.concat(val), []);
};

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
    const issueValues = flatten(projectKeys.map((project) => getIssueValues(metadata, project))).filter(Boolean);

    const issueTypeHash = {};
    issueValues.forEach((issueType) => {
        issueTypeHash[issueType.value] = issueType;
    });

    return Object.values(issueTypeHash);
}

export function getFields(metadata, projectKeys, issueTypeIds) {
    if (!metadata || !projectKeys || !projectKeys.length || !issueTypeIds || !issueTypeIds.length) {
        return {};
    }

    const issueTypesPerProject = projectKeys.map((key) => getIssueTypes(metadata, key).filter((issueType) => issueTypeIds.includes(issueType.id)));

    const fieldsPerProject = [];
    for (const issueTypes of issueTypesPerProject) {
        const projectFields = flatten(issueTypes.map((issueType) =>
            Object.keys(issueType.fields).map((key) => ({...issueType.fields[key], key}))
        )).filter(Boolean);
        fieldsPerProject.push(projectFields);
    }

    // Gather fields
    const fieldHash = {};
    for (const fields of fieldsPerProject) {
        for (const field of fields) {
            fieldHash[field.key] = field;
        }
    }

    // Only keep fields that exist in all selected projects
    for (const fields of fieldsPerProject) {
        for (const fieldId of Object.keys(fieldHash)) {
            if (!fields.find((f) => f.key === fieldId)) {
                delete fieldHash[fieldId];
            }
        }
    }

    return fieldHash;
}

export function getCustomFieldValuesForProjects(metadata, projectKeys) {
    if (!metadata || !projectKeys || !projectKeys.length) {
        return [];
    }

    const issueTypes = flatten(projectKeys.map((key) => getIssueTypes(metadata, key)));

    const customFieldHash = {};
    const fields = flatten(issueTypes.map((issueType) =>
        Object.keys(issueType.fields).map((key) => ({...issueType.fields[key], key}))
    )).filter(Boolean);

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
