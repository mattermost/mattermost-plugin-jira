import {IssueMetadata, JiraField} from 'types/model';

export const useFieldForIssueMetadata = (field: JiraField, key: string): IssueMetadata => {
    return {
        projects: [
            {
                key: 'TEST',
                issuetypes: [{
                    id: '10001',
                    subtask: false,
                    name: 'Bug',
                    fields: {
                        [key]: field,
                    },
                }],
            },
        ],
        issue_types_with_statuses: [
            {
                id: '10001',
                name: 'Bug',
                statuses: [
                    {
                        id: '1001',
                        name: 'TODO',
                    },
                ],
            },
        ],
    };
};
