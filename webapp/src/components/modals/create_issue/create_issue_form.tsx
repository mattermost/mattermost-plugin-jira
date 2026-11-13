// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {Button} from '@mattermost/compass-ui/components/button';

import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';

import {Post} from '@mattermost/types/posts';
import {Team} from '@mattermost/types/teams';

import ConfirmModal from 'components/confirm_modal';
import Modal from 'components/modal';

import {
    APIResponse,
    CreateIssueFields,
    CreateIssueRequest,
    IssueMetadata,
    JiraField,
    SavedFieldValues,
} from 'types/model';

import {getFields, getIssueTypes} from 'utils/jira_issue_metadata';

import Validator from 'components/validator';
import JiraFields from 'components/jira_fields';
import {isFieldSupported} from 'components/jira_field';
import Loading from 'components/loading';
import SelectSetting from 'components/select_setting';
import JiraInstanceAndProjectSelector from 'components/jira_instance_and_project_selector';

type Props = {
    close: (e?: Event) => void;
    create: (issue: CreateIssueRequest) => Promise<APIResponse<{}>>;
    description?: string;
    channelId?: string;
    currentTeam: Team;
    post?: Post;
    visible: boolean;
    fetchJiraIssueMetadataForProjects: (projectKeys: string[], instanceID: string) => Promise<APIResponse<IssueMetadata>>;
};

const FORM_ID = 'jira-create-issue-form';

// Project and issue type are prefilled from saved defaults, so only the issue's own fields count as edits.
const contentSnapshot = (fields: CreateIssueFields): string => JSON.stringify(
    Object.keys(fields).sort()
        .filter((key) => key !== 'project' && key !== 'issuetype' && typeof fields[key] !== 'undefined')
        .map((key) => [key, fields[key]]),
);

type State = {
    submitting: boolean;
    fields: CreateIssueFields;
    instanceID: string | null;
    projectKey: string | null;
    issueType: string | null;
    error: string | null;
    jiraIssueMetadata: IssueMetadata | null;
    fetchingIssueMetadata: boolean;
    showDiscardConfirm: boolean;
};

export default class CreateIssueForm extends React.PureComponent<Props, State> {
    private validator: Validator = new Validator();
    private initialContent: string;
    constructor(props: Props) {
        super(props);

        let description = this.props.description || '';
        if (props.post) {
            description = props.post.message;
        }

        this.state = {
            instanceID: null,
            projectKey: null,
            issueType: null,
            error: null,
            fetchingIssueMetadata: false,
            jiraIssueMetadata: null,
            submitting: false,
            showDiscardConfirm: false,
            fields: {
                description,
                project: {
                    key: '',
                },
                issuetype: {
                    id: '',
                },
            } as CreateIssueFields,
        };
        this.initialContent = contentSnapshot(this.state.fields);
    }

    requestClose = () => {
        if (contentSnapshot(this.state.fields) === this.initialContent) {
            this.props.close();
        } else {
            this.setState({showDiscardConfirm: true});
        }
    };

    handleClose = (e?: React.SyntheticEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.close();
    };

    handleInstanceChange = (instanceID: string) => {
        this.setState({instanceID, projectKey: '', error: null});
    };

    handleProjectChange = (fieldValues: SavedFieldValues) => {
        const projectKey = fieldValues.project_key ? fieldValues.project_key : '';
        this.setState({projectKey, fetchingIssueMetadata: true, error: null});

        this.props.fetchJiraIssueMetadataForProjects([projectKey], this.state.instanceID as string).then(({data, error}) => {
            const state = {
                fetchingIssueMetadata: false,
                error: null,
                jiraIssueMetadata: data,
            } as State;

            if (error) {
                state.error = error.message;
            }

            this.setState(state);
        });

        const fields = {
            summary: this.state.fields.summary,
            description: this.state.fields.description,
            project: {key: projectKey},
        } as CreateIssueFields;

        if (fieldValues.issue_type) {
            fields.issuetype = {
                id: fieldValues.issue_type,
            };
        } else {
            const issueTypes = getIssueTypes(this.state.jiraIssueMetadata, projectKey, {includeSubtasks: false});
            const issueType = issueTypes.length ? issueTypes[0].id : '';
            fields.issuetype = {
                id: issueType,
            };
        }

        this.setState({
            projectKey,
            issueType: fieldValues.issue_type ? fieldValues.issue_type : '',
            fields,
        });
    };

    handleProjectFetchError = (error: string) => {
        this.setState({error});
    };

    handleIssueTypeChange = (_: string, issueType: string) => {
        const fields = {
            summary: this.state.fields.summary,
            description: this.state.fields.description,
            project: {key: this.state.projectKey},
            issuetype: {id: issueType},
        } as CreateIssueFields;

        this.setState({
            issueType,
            fields,
        });
    };

    handleFieldChange = (key: string, value: JiraField) => {
        const fields = {...this.state.fields};
        if (value) {
            if (typeof value === 'object' && 'id' in value && ((value as {id: string}).id === null)) {
                delete fields[key];
            } else {
                fields[key] = value;
            }
        } else {
            delete fields[key];
        }

        this.setState({
            fields,
        });
    };

    getFieldsNotCovered = () => {
        const fields = getFields(
            this.state.jiraIssueMetadata,
            this.state.projectKey,
            this.state.issueType,
        );

        const fieldsNotCovered: [string, string][] = [];
        Object.keys(fields).forEach((key) => {
            const field = fields[key];
            if (field.required) {
                // Field is required and not supported by this modal.
                if (!isFieldSupported(field)) {
                    if (!fieldsNotCovered.find((f) => f[0] === key)) {
                        fieldsNotCovered.push([key, field.name]);
                    }
                }
            }
        });
        return fieldsNotCovered;
    };

    handleSubmit = (e?: React.FormEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        if (!this.validator.validate()) {
            return;
        }

        const {post} = this.props;
        const postId = post ? post.id : '';

        let channelId = this.props.channelId;
        if (post) {
            channelId = post.channel_id;
        }

        const requiredFieldsNotCovered = this.getFieldsNotCovered();
        const issue = {
            post_id: postId,
            current_team: this.props.currentTeam.name,
            fields: this.state.fields,
            channel_id: channelId as string,
            instance_id: this.state.instanceID as string,
            required_fields_not_covered: requiredFieldsNotCovered,
        };

        this.setState({submitting: true});
        this.props.create(issue).then(({error}) => {
            if (error) {
                if (requiredFieldsNotCovered.length && error.message.includes('required fields')) {
                    this.handleClose();
                    return;
                }
                this.setState({error: error.message, submitting: false});
                return;
            }

            this.handleClose();
        });
    };

    renderForm = () => {
        const issueTypes = getIssueTypes(this.state.jiraIssueMetadata, this.state.projectKey, {includeSubtasks: false});
        const issueOptions = issueTypes.map((it) => ({label: it.name, value: it.id}));

        return (
            <div>
                <SelectSetting
                    name={'issue_type'}
                    label={'Issue Type'}
                    required={true}
                    onChange={this.handleIssueTypeChange}
                    options={issueOptions}
                    isMulti={false}
                    value={issueOptions.find((option) => option.value === this.state.issueType)}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                />
                <JiraFields
                    fields={getFields(
                        this.state.jiraIssueMetadata,
                        this.state.projectKey,
                        this.state.issueType,
                    )}
                    instanceID={this.state.instanceID as string}
                    issueMetadata={this.state.jiraIssueMetadata as IssueMetadata}
                    onChange={this.handleFieldChange}
                    values={this.state.fields}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                    isFilter={false}
                />
            </div>
        );
    };

    render() {
        const instanceSelector = (
            <JiraInstanceAndProjectSelector
                selectedInstanceID={this.state.instanceID}
                selectedProjectID={this.state.projectKey}
                onInstanceChange={this.handleInstanceChange}
                onProjectChange={this.handleProjectChange}
                onError={this.handleProjectFetchError}
                addValidate={this.validator.addComponent}
                removeValidate={this.validator.removeComponent}
            />
        );

        const disableSubmit = this.state.fetchingIssueMetadata || !(this.state.projectKey && this.state.jiraIssueMetadata);
        const footer = (
            <React.Fragment>
                <Button
                    emphasis='tertiary'
                    onClick={this.requestClose}
                >
                    {'Cancel'}
                </Button>
                <Button
                    id='submit-button'
                    type='submit'
                    form={FORM_ID}
                    emphasis='primary'
                    loading={this.state.submitting}
                    disabled={disableSubmit}
                >
                    {'Create'}
                </Button>
            </React.Fragment>
        );

        let form;
        if (this.state.fetchingIssueMetadata) {
            form = <Loading/>;
        } else if (this.state.projectKey && this.state.jiraIssueMetadata && this.state.instanceID) {
            form = this.renderForm();
        }

        let error;
        if (this.state.error) {
            error = (
                <SectionNotice
                    className='jira-form-notice'
                    role='alert'
                    type='danger'
                    title={this.state.error}
                />
            );
        }

        return (
            <Modal
                open={this.props.visible}
                size='large'
                title='Create Jira Issue'
                onClose={this.requestClose}
                footer={footer}
            >
                <form
                    id={FORM_ID}
                    role='form'
                    onSubmit={this.handleSubmit}
                >
                    {error}
                    {instanceSelector}
                    {form}
                </form>
                <ConfirmModal
                    show={this.state.showDiscardConfirm}
                    title='Discard changes'
                    message='Are you sure you want to discard your changes?'
                    confirmButtonText='Discard'
                    destructive={true}
                    onCancel={() => this.setState({showDiscardConfirm: false})}
                    onConfirm={() => this.setState({showDiscardConfirm: false}, this.props.close)}
                />
            </Modal>
        );
    }
}
