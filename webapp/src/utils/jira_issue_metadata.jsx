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

export function getFields(metadata, projectKey, issueTypeId) {
    if (!metadata || !projectKey || !issueTypeId) {
        return [];
    }

    return getIssueTypes(metadata, projectKey).find((issueType) => issueType.id === issueTypeId).fields;
}

export function getCustomFieldValuesForProjects(metadata, projectKeys) {
    if (!metadata || !projectKeys || !projectKeys.length) {
        return [];
    }

    const issueTypes = flatten(projectKeys.map((key) => getIssueTypes(metadata, key)));

    const customFieldHash = {};
    const fields = flatten(issueTypes.map((issueType) => Object.values(issueType.fields))).filter(Boolean);

    for (const field of fields) {
        if (field.schema.custom) {
            // Jira server webhook fields don't have keys
            // name is the most unique property available in that case
            const id = field.key || field.name;
            customFieldHash[id] = {...field, id};
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
        value: `event_updated_${field.id}`,
    }));
}

export function getFieldValues(metadata, projectKey, issueTypeId) {
    const fieldsForIssue = getFields(metadata, projectKey, issueTypeId);
    const fieldIds = Object.keys(fieldsForIssue);
    return fieldIds.map((fieldId) => ({value: fieldId, label: fieldsForIssue[fieldId].name}));
}
