// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Modal} from 'react-bootstrap';

import {Theme} from 'mattermost-redux/types/preferences';
import {Post} from 'mattermost-redux/types/posts';
import {Team} from 'mattermost-redux/types/teams';

import {APIResponse, IssueMetadata, CreateIssueRequest, JiraFieldTypeEnums, JiraFieldCustomTypeEnums, CreateIssueFields, JiraField, SavedFieldValues} from 'types/model';

import {getFields, getIssueTypes} from 'utils/jira_issue_metadata';
import {getModalStyles} from 'utils/styles';

import Validator from 'components/validator';
import JiraFields from 'components/jira_fields';
import FormButton from 'components/form_button';
import Loading from 'components/loading';
import ReactSelectSetting from 'components/react_select_setting';
import JiraInstanceAndProjectSelector from 'components/jira_instance_and_project_selector';

const allowedFields: string[] = [
    JiraFieldTypeEnums.PROJECT,
    JiraFieldTypeEnums.ISSUE_TYPE,
    JiraFieldTypeEnums.PRIORITY,
    JiraFieldTypeEnums.DESCRIPTION,
    JiraFieldTypeEnums.SUMMARY,
    JiraFieldTypeEnums.LABELS,
    JiraFieldTypeEnums.ASSIGNEE,
    JiraFieldTypeEnums.SECURITY,
    JiraFieldTypeEnums.COMPONENTS,
    JiraFieldTypeEnums.FIX_VERSIONS,
    JiraFieldTypeEnums.AFFECTS_VERSIONS,
];

const allowedSchemaCustom: string[] = [
    JiraFieldCustomTypeEnums.TEXT_AREA,
    JiraFieldCustomTypeEnums.TEXT_FIELD,
    JiraFieldCustomTypeEnums.SELECT,
    JiraFieldCustomTypeEnums.PROJECT,
    JiraFieldCustomTypeEnums.EPIC_LINK,
    JiraFieldCustomTypeEnums.EPIC_NAME,
    JiraFieldCustomTypeEnums.CASCADING_SELECT,
    JiraFieldCustomTypeEnums.MULTI_SELECT,
    JiraFieldCustomTypeEnums.RADIO_BUTTONS,
    JiraFieldCustomTypeEnums.MULTI_CHECKBOXES,
    JiraFieldCustomTypeEnums.URL_FIELD,
    JiraFieldCustomTypeEnums.LABELS,
    JiraFieldCustomTypeEnums.USER_PICKER,
];

type Props = {
    close: (e?: Event) => void;
    create: (issue: CreateIssueRequest) => Promise<APIResponse<{}>>;
    description?: string;
    channelId?: string;
    currentTeam: Team;
    post?: Post;
    theme: Theme;
    visible: boolean;
    fetchJiraIssueMetadataForProjects: (projectKeys: string[], instanceID: string) => Promise<APIResponse<IssueMetadata>>;
};

type State = {
    submitting: boolean;
    fields: CreateIssueFields;
    instanceID: string | null;
    projectKey: string | null;
    issueType: string | null;
    error: string | null;
    jiraIssueMetadata: IssueMetadata | null;
    fetchingIssueMetadata: boolean;
};

export default class CreateIssueForm extends React.PureComponent<Props, State> {
    private validator: Validator = new Validator();
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
    }

    handleClose = (e?: Event) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.close();
    }

    handleInstanceChange = (instanceID: string) => {
        this.setState({instanceID, projectKey: '', error: null});
    }

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
            const issueTypes = getIssueTypes(this.state.jiraIssueMetadata, projectKey);
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
    }

    handleProjectFetchError = (error: string) => {
        this.setState({error});
    }

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
    }

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
    }

    getFieldsNotCovered = () => {
        const fields = getFields(
            this.state.jiraIssueMetadata,
            this.state.projectKey,
            this.state.issueType
        );

        const fieldsNotCovered: [string, string][] = [];
        Object.keys(fields).forEach((key) => {
            const field = fields[key];
            if (field.required) {
                // Field is required and not supported by this modal.
                if ((!field.schema.custom && !allowedFields.includes(key)) || (field.schema.custom && !allowedSchemaCustom.includes(field.schema.custom))) {
                    if (!fieldsNotCovered.find((f) => f[0] === key)) {
                        fieldsNotCovered.push([key, field.name]);
                    }
                }
            }
        });
        return fieldsNotCovered;
    }

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
    }

    renderForm = () => {
        const issueTypes = getIssueTypes(this.state.jiraIssueMetadata, this.state.projectKey);
        const issueOptions = issueTypes.map((it) => ({label: it.name, value: it.id}));

        return (
            <div>
                <ReactSelectSetting
                    name={'issue_type'}
                    label={'Issue Type'}
                    required={true}
                    onChange={this.handleIssueTypeChange}
                    options={issueOptions}
                    isMulti={false}
                    theme={this.props.theme}
                    value={issueOptions.find((option) => option.value === this.state.issueType)}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                />
                <JiraFields
                    fields={getFields(
                        this.state.jiraIssueMetadata,
                        this.state.projectKey,
                        this.state.issueType
                    )}
                    instanceID={this.state.instanceID as string}
                    issueMetadata={this.state.jiraIssueMetadata as IssueMetadata}
                    onChange={this.handleFieldChange}
                    values={this.state.fields}
                    allowedFields={allowedFields}
                    allowedSchemaCustom={allowedSchemaCustom}
                    theme={this.props.theme}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                    isFilter={false}
                />
            </div>
        );
    }

    render() {
        const style = getModalStyles(this.props.theme);

        const instanceSelector = (
            <JiraInstanceAndProjectSelector
                selectedInstanceID={this.state.instanceID}
                selectedProjectID={this.state.projectKey}
                onInstanceChange={this.handleInstanceChange}
                onProjectChange={this.handleProjectChange}
                onError={this.handleProjectFetchError}

                theme={this.props.theme}
                addValidate={this.validator.addComponent}
                removeValidate={this.validator.removeComponent}
            />
        );

        const disableSubmit = this.state.fetchingIssueMetadata || !(this.state.projectKey && this.state.jiraIssueMetadata);
        const footer = (
            <React.Fragment>
                <FormButton
                    type='button'
                    btnClass='btn-link'
                    defaultMessage='Cancel'
                    onClick={this.handleClose}
                />
                <FormButton
                    id='submit-button'
                    type='submit'
                    btnClass='btn btn-primary'
                    saving={this.state.submitting}
                    disabled={disableSubmit}
                >
                    {'Create'}
                </FormButton>
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
                <p className='alert alert-danger'>
                    <i
                        style={{marginRight: '10px'}}
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span>{this.state.error}</span>
                </p>
            );
        }

        return (
            <form
                role='form'
                onSubmit={this.handleSubmit}
            >
                <Modal.Body
                    style={style.modalBody}
                >
                    {error}
                    {instanceSelector}
                    {form}
                </Modal.Body>
                <Modal.Footer style={style.modalFooter}>
                    {footer}
                </Modal.Footer>
            </form>
        );
    }
}
