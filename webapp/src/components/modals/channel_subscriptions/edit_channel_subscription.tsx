// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {Button} from '@mattermost/compass-ui/components/button';
import {ErrorMessage} from '@mattermost/compass-ui/components/error-message';
import {Modal} from '@mattermost/compass-ui/components/modal';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';

import SelectSetting from 'components/select_setting';
import ConfirmModal from 'components/confirm_modal';
import Input from 'components/input';
import Loading from 'components/loading';
import Validator from 'components/validator';
import JiraInstanceAndProjectSelector from 'components/jira_instance_and_project_selector';

import {
    filterValueIsSecurityField,
    generateJQLStringFromSubscriptionFilters,
    getConflictingFields,
    getCustomFieldFiltersForProjects,
    getCustomFieldValuesForEvents,
    getIssueTypes,
} from 'utils/jira_issue_metadata';

import {
    ChannelSubscription,
    ChannelSubscriptionFilters as ChannelSubscriptionFiltersModel,
    FilterValue,
    IssueMetadata,
    SavedFieldValues,
    SelectOption,
} from 'types/model';

import ChannelSubscriptionFilters from './channel_subscription_filters';
import {SharedProps} from './shared_props';

const JiraEventOptions: SelectOption[] = [
    {value: 'event_created', label: 'Issue Created'},
    {value: 'event_deleted', label: 'Issue Deleted'},
    {value: 'event_deleted_unresolved', label: 'Issue Deleted, Unresolved'},
    {value: 'event_updated_reopened', label: 'Issue Reopened'},
    {value: 'event_updated_resolved', label: 'Issue Resolved'},
    {value: 'event_created_comment', label: 'Comment Created'},
    {value: 'event_updated_comment', label: 'Comment Updated'},
    {value: 'event_deleted_comment', label: 'Comment Deleted'},
    {value: 'event_updated_any', label: 'Issue Updated: Any'},
    {value: 'event_updated_affects_version', label: 'Issue Updated: Affects Version'},
    {value: 'event_updated_assignee', label: 'Issue Updated: Assignee'},
    {value: 'event_updated_attachment', label: 'Issue Updated: Attachment'},
    {value: 'event_updated_description', label: 'Issue Updated: Description'},
    {value: 'event_updated_fix_version', label: 'Issue Updated: Fix Version'},
    {value: 'event_updated_issue_type', label: 'Issue Updated: Issue Type'},
    {value: 'event_updated_labels', label: 'Issue Updated: Labels'},
    {value: 'event_updated_priority', label: 'Issue Updated: Priority'},
    {value: 'event_updated_rank', label: 'Issue Updated: Rank'},
    {value: 'event_updated_reporter', label: 'Issue Updated: Reporter'},
    {value: 'event_updated_sprint', label: 'Issue Updated: Sprint'},
    {value: 'event_updated_status', label: 'Issue Updated: Status'},
    {value: 'event_updated_summary', label: 'Issue Updated: Summary'},
    {value: 'event_updated_components', label: 'Issue Updated: Components'},
];

export type Props = SharedProps & {
    finishEditSubscription: () => void;
    selectedSubscription: ChannelSubscription | null;
    creatingSubscription: boolean;
    creatingSubscriptionTemplate: boolean;
    selectedSubscriptionTemplate: ChannelSubscription | null;
};

export type State = {
    filters: ChannelSubscriptionFiltersModel;
    instanceID: string;
    fetchingIssueMetadata: boolean;
    jiraIssueMetadata: IssueMetadata | null;
    templateOptions: SelectOption[] | null;
    error: string | null;
    getMetaDataErr: string | null;
    submitting: boolean;
    submittingTemplate: boolean;
    subscriptionName: string | null;
    showConfirmModal: boolean;
    confirmActionType: 'delete' | 'close' | null;
    conflictingError: string | null;
    selectedTemplateID: string | null;
};

export default class EditChannelSubscription extends PureComponent<Props, State> {
    private validator: Validator;

    constructor(props: Props) {
        super(props);

        let filters: ChannelSubscriptionFiltersModel = {
            events: [],
            projects: [],
            issue_types: [],
            fields: [],
        };

        let subscriptionName = null;
        if (props.selectedSubscription) {
            filters = Object.assign({}, filters, props.selectedSubscription.filters);
            subscriptionName = props.selectedSubscription.name;
        }

        if (props.selectedSubscriptionTemplate) {
            filters = Object.assign({}, filters, props.selectedSubscriptionTemplate.filters);
            subscriptionName = props.selectedSubscriptionTemplate.name;
        }

        filters.fields = filters.fields || [];

        let instanceID = '';
        let fetchingIssueMetadata = false;
        if (this.props.selectedSubscription) {
            instanceID = this.props.selectedSubscription.instance_id;
        }

        if (this.props.selectedSubscriptionTemplate) {
            instanceID = this.props.selectedSubscriptionTemplate.instance_id;
        }

        if (filters.projects.length && instanceID) {
            fetchingIssueMetadata = true;
            this.fetchIssueMetadata(filters.projects, instanceID);
        }

        this.state = {
            error: null,
            getMetaDataErr: null,
            submitting: false,
            submittingTemplate: false,
            filters,
            fetchingIssueMetadata,
            jiraIssueMetadata: null,
            subscriptionName,
            showConfirmModal: false,
            confirmActionType: null,
            conflictingError: null,
            instanceID,
            selectedTemplateID: null,
            templateOptions: null,
        };

        this.validator = new Validator();
    }

    componentDidMount() {
        if (this.props.selectedSubscription) {
            const projects = this.props.selectedSubscription.filters.projects;
            if (projects.length) {
                this.fetchSubscriptionTemplateForProjectKey(this.state.instanceID, projects[0]);
            }
        }

        if (this.props.selectedSubscriptionTemplate) {
            const projects = this.props.selectedSubscriptionTemplate.filters.projects;
            if (projects.length) {
                this.fetchSubscriptionTemplateForProjectKey(this.state.instanceID, projects[0]);
            }
        }
    }

    handleCancel = (): void => {
        this.setState({showConfirmModal: true, confirmActionType: 'close'});
    };

    handleClose = (e?: React.FormEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.finishEditSubscription();
    };

    handleNameChange = (id: string, value: string) => {
        this.setState({subscriptionName: value});
    };

    deleteChannelSubscription = () => {
        if (this.props.selectedSubscription) {
            this.props.deleteChannelSubscription(this.props.selectedSubscription).then((res) => {
                if (res.error) {
                    this.setState({error: res.error.message});
                } else {
                    this.handleClose();
                }
            });
        }

        if (this.props.selectedSubscriptionTemplate) {
            this.props.deleteSubscriptionTemplate(this.props.selectedSubscriptionTemplate).then((res) => {
                if (res.error) {
                    this.setState({error: res.error.message});
                } else {
                    this.handleClose();
                }
            });
        }
    };

    handleCancelAction = () => {
        this.setState({showConfirmModal: false});
    };

    handleConfirmAction = () => {
        if (this.state.confirmActionType === 'close') {
            this.props.finishEditSubscription();
        } else if (this.state.confirmActionType === 'delete') {
            this.deleteChannelSubscription();
        }

        this.setState({showConfirmModal: false, confirmActionType: null});
    };

    handleDeleteChannelSubscription = (): void => {
        this.setState({showConfirmModal: true, confirmActionType: 'delete'});
    };

    handleSettingChange = (id: keyof ChannelSubscriptionFiltersModel, value: string[]) => {
        let finalValue = value;
        if (!finalValue) {
            finalValue = [];
        } else if (!Array.isArray(finalValue)) {
            finalValue = [finalValue];
        }
        const filters = {...this.state.filters};
        filters[id] = finalValue;
        this.setState({filters});
        this.clearConflictingErrorMessage();
    };

    clearConflictingErrorMessage = () => {
        this.setState({conflictingError: null});
    };

    shouldShowEmptySecurityLevelMessage = (): boolean => {
        if (!this.props.securityLevelEmptyForJiraSubscriptions) {
            return false;
        }

        return !this.state.filters.fields.some(filterValueIsSecurityField);
    };

    handleIssueChange = (id: keyof ChannelSubscriptionFiltersModel, value: string[] | null) => {
        const finalValue = value || [];
        const filters = {...this.state.filters, issue_types: finalValue};

        let conflictingFields = null;
        if (finalValue.length > this.state.filters.issue_types.length) {
            const filterFields = getCustomFieldFiltersForProjects(this.state.jiraIssueMetadata, this.state.filters.projects, this.state.filters.issue_types);
            conflictingFields = getConflictingFields(
                filterFields,
                finalValue,
                this.state.jiraIssueMetadata,
            );
        }

        if (conflictingFields && conflictingFields.length) {
            const selectedConflictingFields = conflictingFields.filter((f1) => {
                return this.state.filters.fields.find((f2) => f1.field.key === f2.key);
            });

            if (selectedConflictingFields.length) {
                const fieldsStr = selectedConflictingFields.map((cf) => cf.field.name).join(', ');
                const conflictingIssueType = conflictingFields[0].issueTypes[0];

                let errorStr = `Issue Type(s) "${conflictingIssueType.name}" does not have filter field(s): "${fieldsStr}".  `;
                errorStr += 'Please update the conflicting fields or create a separate subscription.';
                this.setState({conflictingError: errorStr});
                return;
            }
        }

        this.setState({filters, conflictingError: null});
    };

    fetchIssueMetadata = (projectKeys: string[], instanceID: string) => {
        if (!instanceID) {
            this.setState({getMetaDataErr: 'No Jira instance is selected.'});
        }

        this.props.fetchJiraIssueMetadataForProjects(projectKeys, instanceID).then(({data, error}) => {
            const jiraIssueMetadata = data as IssueMetadata;
            const state = {fetchingIssueMetadata: false, jiraIssueMetadata} as State;

            if (error) {
                state.getMetaDataErr = `The project ${projectKeys[0]} is unavailable. Please contact your system administrator.`;
            }

            const filterFields = getCustomFieldFiltersForProjects(jiraIssueMetadata, this.state.filters.projects, this.state.filters.issue_types);
            for (const v of this.state.filters.fields) {
                if (!filterFields.find((f) => f.key === v.key)) {
                    state.error = 'A field in this subscription has been removed from Jira, so the subscription is invalid. When this form is submitted, the configured field will be removed from the subscription to make the subscription valid again.';
                }
            }

            this.setState(state);
        });
    };

    fetchSubscriptionTemplateForProjectKey = (instanceId: string, projectId: string) => {
        this.setState({selectedTemplateID: null, fetchingIssueMetadata: true});
        this.props.fetchSubscriptionTemplatesForProjectKey(instanceId, projectId, this.props.channel.id).then((subs) => {
            if (subs.error) {
                this.setState({error: subs.error.message});
                return;
            }

            const subscriptionTemplate = subs.data as ChannelSubscription[];
            let templateOptions: SelectOption[] | null = null;
            if (subscriptionTemplate) {
                templateOptions = subscriptionTemplate.map((template: ChannelSubscription) => (
                    {label: template.name || template.id, value: template.id}
                ));
            }

            this.setState({templateOptions, fetchingIssueMetadata: false});
        });
    };

    handleJiraInstanceChange = (instanceID: string) => {
        if (instanceID === this.state.instanceID) {
            return;
        }

        this.setState({instanceID, error: null});
        this.handleProjectChange({});
    };

    handleProjectChange = (fieldValues: SavedFieldValues) => {
        const projectID = fieldValues.project_key ? fieldValues.project_key : '';
        this.clearConflictingErrorMessage();

        let projects: string[];
        if (projectID) {
            projects = [projectID];
        } else {
            projects = [];
        }

        if (projects.length && this.state.filters.projects[0] === projects[0]) {
            return;
        }

        const filters = {
            projects,
            issue_types: [],
            events: [],
            fields: [],
        };

        let fetchingIssueMetadata = false;

        if (projects && projects.length) {
            fetchingIssueMetadata = true;
            this.fetchIssueMetadata(projects, this.state.instanceID);
        }

        if (this.state.instanceID && projectID) {
            fetchingIssueMetadata = true;
            this.fetchSubscriptionTemplateForProjectKey(this.state.instanceID, projectID);
        }

        this.setState({
            fetchingIssueMetadata,
            getMetaDataErr: null,
            filters,
        });
    };

    handleFilterFieldChange = (fields: FilterValue[]) => {
        this.setState({filters: {...this.state.filters, fields}});
        this.clearConflictingErrorMessage();
    };

    handleCreate = (e?: React.FormEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        if (!this.validator.validate()) {
            return;
        }

        if (!this.state.subscriptionName || this.state.subscriptionName.trim() === '') {
            this.setState({error: 'Name cannot be empty or only whitespaces.'});
            return;
        }

        const filterFields = getCustomFieldFiltersForProjects(this.state.jiraIssueMetadata, this.state.filters.projects, this.state.filters.issue_types);
        const configuredFields = this.state.filters.fields.concat([]);
        for (const v of this.state.filters.fields) {
            if (!filterFields.find((f) => f.key === v.key)) {
                configuredFields.splice(configuredFields.indexOf(v), 1);
            }
        }

        const filters = {
            ...this.state.filters,
            fields: configuredFields,
        };

        const subscription = {
            channel_id: this.props.channel.id,
            filters,
            name: this.state.subscriptionName?.trim(),
            instance_id: this.state.instanceID,
        } as ChannelSubscription;

        if (this.props.selectedSubscriptionTemplate) {
            this.setState({submittingTemplate: true, error: null});
            subscription.id = this.props.selectedSubscriptionTemplate.id;
            this.props.editSubscriptionTemplate(subscription).then((edited) => {
                if (edited.error) {
                    this.setState({error: edited.error.message, submittingTemplate: false});
                    return;
                }

                this.handleClose(e);
            });
        } else if (this.props.creatingSubscriptionTemplate) {
            this.setState({submittingTemplate: true, error: null});
            this.props.createSubscriptionTemplate(subscription).then((created) => {
                if (created.error) {
                    this.setState({error: created.error.message, submittingTemplate: false});
                    return;
                }

                this.handleClose(e);
            });
        } else if (this.props.selectedSubscription) {
            this.setState({submitting: true, error: null});
            subscription.id = this.props.selectedSubscription.id;
            this.props.editChannelSubscription(subscription).then((edited) => {
                if (edited.error) {
                    this.setState({error: edited.error.message, submitting: false});
                    return;
                }
                this.handleClose(e);
            });
        } else {
            this.setState({submitting: true, error: null});
            this.props.createChannelSubscription(subscription).then((created) => {
                if (created.error) {
                    this.setState({error: created.error.message, submitting: false});
                    return;
                }
                this.handleClose(e);
            });
        }
    };

    handleTemplateChange = (_: any, templateId: string) => {
        const templateChoosen = this.props.subscriptionTemplates.find((template) => template.id === templateId);
        this.handleProjectChange(templateChoosen.filters.projects[0]);
        this.setState({
            filters: templateChoosen.filters,
            selectedTemplateID: templateId,
        });
    };

    render(): JSX.Element {
        const issueTypes = getIssueTypes(this.state.jiraIssueMetadata, this.state.filters.projects[0], {includeSubtasks: true});
        const issueOptions = issueTypes.map((it) => ({label: it.name, value: it.id}));

        const customFields = getCustomFieldValuesForEvents(this.state.jiraIssueMetadata, this.state.filters.projects);
        const filterFields = getCustomFieldFiltersForProjects(this.state.jiraIssueMetadata, this.state.filters.projects, this.state.filters.issue_types);

        const eventOptions = JiraEventOptions.concat(customFields);

        let conflictingErrorComponent = null;
        if (this.state.conflictingError) {
            conflictingErrorComponent = (
                <ErrorMessage
                    className='jira-form-notice'
                    message={this.state.conflictingError}
                />
            );
        }

        let component = null;
        if (this.props.channel && this.props.channelSubscriptions) {
            let innerComponent = null;
            if (this.state.fetchingIssueMetadata) {
                innerComponent = <Loading/>;
            } else if (this.state.filters.projects[0] && !this.state.getMetaDataErr && this.state.jiraIssueMetadata) {
                innerComponent = (
                    <React.Fragment>
                        <SelectSetting
                            name='template'
                            label='Use Template'
                            options={this.state.templateOptions}
                            onChange={this.handleTemplateChange}
                            value={this.state.templateOptions && this.state.templateOptions.find((option) => option.value === this.state.selectedTemplateID)}
                            required={false}
                            isLoading={false}
                        />
                        <SelectSetting
                            name='events'
                            label='Events'
                            required={true}
                            onChange={this.handleSettingChange}
                            options={eventOptions}
                            isMulti={true}
                            value={eventOptions.filter((option) => this.state.filters.events.includes(option.value))}
                            addValidate={this.validator.addComponent}
                            removeValidate={this.validator.removeComponent}
                        />
                        <SelectSetting
                            name='issue_types'
                            label='Issue Type'
                            required={true}
                            onChange={this.handleIssueChange}
                            options={issueOptions}
                            isMulti={true}
                            value={issueOptions.filter((option) => this.state.filters.issue_types.includes(option.value))}
                            addValidate={this.validator.addComponent}
                            removeValidate={this.validator.removeComponent}
                        />
                        {conflictingErrorComponent}
                        <ChannelSubscriptionFilters
                            fields={filterFields}
                            values={this.state.filters.fields}
                            chosenIssueTypes={this.state.filters.issue_types}
                            issueMetadata={this.state.jiraIssueMetadata}
                            onChange={this.handleFilterFieldChange}
                            addValidate={this.validator.addComponent}
                            removeValidate={this.validator.removeComponent}
                            instanceID={this.state.instanceID}
                            securityLevelEmptyForJiraSubscriptions={this.props.securityLevelEmptyForJiraSubscriptions}
                            searchTeamFields={this.props.searchTeamFields}
                            projectKey={this.state.filters.projects[0] || ''}
                        />
                        <div className='jira-subscriptions__jql'>
                            <h4 className='jira-subscriptions__jql-label'>
                                {'Approximate JQL Output'}
                            </h4>
                            <code className='jira-subscriptions__jql-code'>
                                {generateJQLStringFromSubscriptionFilters(this.state.jiraIssueMetadata, filterFields, this.state.filters, this.props.securityLevelEmptyForJiraSubscriptions)}
                            </code>
                            {this.shouldShowEmptySecurityLevelMessage() && (
                                <SectionNotice
                                    type='info'
                                    title='No security level filter selected'
                                    description='The subscription will only allow issues that have no security level assigned.'
                                />
                            )}
                            <Button
                                asChild={true}
                                emphasis='link'
                                size='small'
                            >
                                <a
                                    href='https://github.com/mattermost/mattermost-plugin-jira#create-a-channel-subscription'
                                    target='_blank'
                                    rel='noopener noreferrer'
                                >
                                    {'Learn More'}
                                </a>
                            </Button>
                        </div>
                    </React.Fragment>
                );
            }

            component = (
                <React.Fragment>
                    <Input
                        label='Subscription Name'
                        placeholder='Name'
                        type={'input'}
                        maxLength={100}
                        required={true}
                        onChange={this.handleNameChange}
                        value={this.state.subscriptionName}
                        readOnly={false}
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                    />
                    <JiraInstanceAndProjectSelector
                        selectedInstanceID={this.state.instanceID}
                        selectedProjectID={this.state.filters.projects[0]}
                        onInstanceChange={this.handleJiraInstanceChange}
                        onProjectChange={this.handleProjectChange}
                        onError={(error: string) => this.setState({error})}
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                    />
                    {innerComponent}
                </React.Fragment>
            );
        } else {
            component = <Loading/>;
        }

        const {showConfirmModal} = this.state;

        let confirmMessage = 'Are you sure you want to discard your changes?';
        if (this.state.confirmActionType === 'delete') {
            confirmMessage = `Are you sure to delete the subscription${this.props.selectedSubscription?.name ? ` "${this.props.selectedSubscription.name}"` : ''}?`;
            if (this.props.selectedSubscriptionTemplate) {
                confirmMessage = `Are you sure to delete the subscription template${this.props.selectedSubscriptionTemplate.name ? ` "${this.props.selectedSubscriptionTemplate.name}"` : ''}?`;
            }
        }

        const isSubscription = Boolean(this.props.selectedSubscription || this.props.creatingSubscription);
        const confirmComponent = (
            <ConfirmModal
                cancelButtonText='Cancel'
                confirmButtonText={this.state.confirmActionType === 'delete' ? 'Delete' : 'Discard'}
                destructive={true}
                hideCancel={false}
                message={confirmMessage}
                onCancel={this.handleCancelAction}
                onConfirm={this.handleConfirmAction}
                show={showConfirmModal}
                title={isSubscription ? 'Subscription' : 'Subscription Template'}
            />
        );

        const error = this.state.error || this.state.getMetaDataErr;

        const enableSubmitButton = Boolean(this.state.filters.projects[0]);
        const enableDeleteButton = Boolean(this.props.selectedSubscription || this.props.selectedSubscriptionTemplate);
        let saveSubscriptionButtonText = '';
        let headerText = '';
        if (isSubscription) {
            saveSubscriptionButtonText = this.props.creatingSubscription ? 'Add Subscription' : 'Save Subscription';
            headerText = this.props.creatingSubscription ? 'Add subscription' : 'Edit subscription';
            if (this.props.channel) {
                headerText += ` in ${this.props.channel.display_name}`;
            }
        } else {
            saveSubscriptionButtonText = 'Add Template';
            headerText = 'Add subscription template';
            if (this.props.selectedSubscriptionTemplate && this.props.selectedSubscriptionTemplate.name) {
                saveSubscriptionButtonText = 'Save Template';
                headerText = 'Edit subscription template';
            }
        }

        const submitting = isSubscription ? this.state.submitting : this.state.submittingTemplate;
        const deleteButton = (
            <Button
                id='jira-delete-subscription'
                emphasis='tertiary'
                destructive={true}
                disabled={!enableDeleteButton}
                onClick={this.handleDeleteChannelSubscription}
            >
                {'Delete'}
            </Button>
        );
        const footer = (
            <>
                <Button
                    emphasis='tertiary'
                    onClick={this.handleCancel}
                >
                    {'Cancel'}
                </Button>
                <Button
                    emphasis='primary'
                    disabled={!enableSubmitButton}
                    loading={submitting}
                    onClick={this.handleCreate}
                >
                    {saveSubscriptionButtonText}
                </Button>
            </>
        );

        return (
            <Modal
                size='large'
                title='Jira Subscriptions'
                subtitle={headerText}
                showBackButton={true}
                onBack={this.handleCancel}
                onClose={this.props.close}
                footer={footer}
                footerLeading={deleteButton}
            >
                <form role='form'>
                    {error && (
                        <SectionNotice
                            className='jira-form-notice'
                            role='alert'
                            type='danger'
                            title={error}
                        />
                    )}
                    {component}
                </form>
                {confirmComponent}
            </Modal>
        );
    }
}
