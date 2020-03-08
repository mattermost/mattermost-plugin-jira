// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Modal} from 'react-bootstrap';

import ReactSelectSetting from 'components/react_select_setting';
import ConfirmModal from 'components/confirm_modal';
import FormButton from 'components/form_button';
import Input from 'components/input';
import Loading from 'components/loading';
import Validator from 'components/validator';
import {getBaseStyles} from 'utils/styles';
import {
    getProjectValues,
    getIssueValuesForMultipleProjects,
    getCustomFieldValuesForEvents,
    getCustomFieldFiltersForProjects,
    getConflictingFields,
    generateJQLStringFromSubscriptionFilters,
} from 'utils/jira_issue_metadata';

import {ChannelSubscription, ChannelSubscriptionFilters, ReactSelectOption, FilterField, IssueTypeIdentifier} from 'types/model';

import ChannelSettingsFilters from './channel_settings_filters';
import {SharedProps} from './shared_props';

const JiraEventOptions: ReactSelectOption[] = [
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

type ProjectStatus = {
    id: string;
    name: string;
}

type ProjectStatusesResponse = {
    data: Array<ProjectStatus>;
    error: Error | null;
}

export type Props = SharedProps & {
    finishEditSubscription: () => void;
    selectedSubscription: ChannelSubscription | null;
    creatingSubscription: boolean;
    fetchJiraProjectStatuses: () => Promise<ProjectStatusesResponse>;
};

export type State = {
    filters: ChannelSubscriptionFilters;
    fetchingIssueMetadata: boolean;
    fetchingProjectStatuses: boolean;
    projectStatusField: FilterField;
    error: string | null;
    getMetaDataErr: string | null;
    submitting: boolean;
    subscriptionName: string | null;
    showConfirmModal: boolean;
    conflictingError: string | null;
};

export default class EditChannelSettings extends PureComponent<Props, State> {
    private validator: Validator;

    constructor(props: Props) {
        super(props);

        let filters: ChannelSubscriptionFilters = {
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

        filters.fields = filters.fields || [];

        let fetchingIssueMetadata = false;
        if (filters.projects.length) {
            fetchingIssueMetadata = true;
            this.fetchIssueMetadata(filters.projects);
        }

        // Prepare to create statuses filter currently empty
        const projectStatusField = {
            key: 'status',
            name: 'Status',
            schema: {
                type: 'array',
                items: 'options',
            },
            userDefined: false,
            issueTypes: [],
        };

        this.state = {
            error: null,
            getMetaDataErr: null,
            submitting: false,
            filters,
            fetchingIssueMetadata,
            fetchingProjectStatuses: false,
            projectStatusField,
            subscriptionName,
            showConfirmModal: false,
            conflictingError: null,
        };

        this.validator = new Validator();
    }

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        this.props.finishEditSubscription();
    };

    handleNameChange = (id, value) => {
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
    };

    handleCancelDelete = () => {
        this.setState({showConfirmModal: false});
    }

    handleConfirmDelete = () => {
        this.setState({showConfirmModal: false});
        this.deleteChannelSubscription();
    }

    handleDeleteChannelSubscription = (): void => {
        this.setState({showConfirmModal: true});
    };

    handleSettingChange = (id: keyof ChannelSubscriptionFilters, value: string[]) => {
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
    }

    handleIssueChange = (id: keyof ChannelSubscriptionFilters, value: string[] | null) => {
        const finalValue = value || [];
        const filters = {...this.state.filters, issue_types: finalValue};

        let conflictingFields = null;
        if (finalValue.length > this.state.filters.issue_types.length) {
            const filterFields = getCustomFieldFiltersForProjects(this.props.jiraIssueMetadata, this.state.filters.projects);
            conflictingFields = getConflictingFields(
                filterFields,
                finalValue,
                this.props.jiraIssueMetadata
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

    fetchIssueMetadata = (projectKeys) => {
        this.props.fetchJiraIssueMetadataForProjects(projectKeys).then((fetched) => {
            const state = {fetchingIssueMetadata: false} as State;

            const error = fetched.error || (fetched.data && fetched.data.error);
            if (error) {
                state.getMetaDataErr = `The project ${projectKeys[0]} is unavailable. Please contact your system administrator.`;
            }

            const filterFields = getCustomFieldFiltersForProjects(this.props.jiraIssueMetadata, this.state.filters.projects);
            for (const v of this.state.filters.fields) {
                if (!filterFields.find((f) => f.key === v.key)) {
                    state.error = 'A field in this subscription has been removed from Jira, so the subscription is invalid. When this form is submitted, the configured field will be removed from the subscription to make the subscription valid again.';
                }
            }

            this.setState(state);
        });
    };

    /**
     * fetchProjectStatuses is internal class function which calls "fetchJiraProjectStatuses" func to get the list of projects statuses
     * and updates the status filter field with those new values
     * @param none
     * @returns none
     */
    fetchProjectStatuses = (): void => {
        const {fetchJiraProjectStatuses} = this.props;

        // API call function which returns back us the list of statuses in promise
        fetchJiraProjectStatuses().then(({data, error}) => {
            // Once promise is resolved, set the enum of fetching statuses to false
            const state = {fetchingProjectStatuses: false} as State;

            //  List of statuses could be empty or non empty, we will populate filter field when its not empty; else leave it as it is.
            const validNonEmptyResponse = data && Array.isArray(data) && data.length !== 0;

            // If error is returned from the response
            if (error) {
                state.error = 'Failed to get JIRA project statuses';
            } else if (validNonEmptyResponse) {
                // JIRA api returns all the statuses in all projects irrespective of repetitive status name. so we filter it
                // as it doesn't matter much for JQL, JQL needs only the status name
                const filteredStatuses = data.filter((element, position, originalArray) => {
                    return originalArray.findIndex((uniqueElement) => uniqueElement.name === element.name) === position;
                });

                // Now sort it for cosmetic reasons
                const sortedFilteredStatuses = filteredStatuses.sort((statusA, statusB) => {
                    const statusNameA = statusA.name.toUpperCase();
                    const statusNameB = statusB.name.toUpperCase();

                    if (statusNameA < statusNameB) {
                        return -1;
                    }
                    if (statusNameA > statusNameB) {
                        return 1;
                    }
                    return 0;
                });

                // Filter field should appear for all issue types of all projects
                const issueOptions = getIssueValuesForMultipleProjects(this.props.jiraProjectMetadata, this.state.filters.projects);

                // Convert it to proper format for React Select component
                const issueTypes: Array<IssueTypeIdentifier> = issueOptions.map((issueOption) => ({id: issueOption.value, name: issueOption.label}));

                // Convert the array of statuses we filtered and sorted to match format of React Select component
                const values: Array<ReactSelectOption> = sortedFilteredStatuses.map((status) => ({label: status.name, value: status.id}));

                // Combine it to status filter field
                const projectStatusField: FilterField = Object.assign(this.state.projectStatusField, {
                    values,
                    issueTypes,
                });

                state.projectStatusField = projectStatusField;
            }

            // Update the state in the end with relevant properties, either error or no error ones
            // This helps use just to write one set state instead of many (based on situations) in above
            this.setState(state);
        });
    }

    handleProjectChange = (id, value) => {
        this.clearConflictingErrorMessage();

        let projects = value;
        if (!projects) {
            projects = [];
        } else if (!Array.isArray(projects)) {
            projects = [projects];
        }

        const filters = {
            projects,
            issue_types: [],
            events: [],
            fields: [],
        };

        let fetchingIssueMetadata = false;

        this.props.clearIssueMetadata();
        if (projects && projects.length) {
            fetchingIssueMetadata = true;
            this.fetchIssueMetadata(projects);
        }

        let fetchingProjectStatuses = false;

        // Jira api collects all the status across all the projects in a single request
        // So here assuming user has at least a single status in the project, if not
        // then we will safely assume the api call was never made and make again.
        const doesProjectHaveNoStatus = (this.state.projectStatusField && this.state.projectStatusField.values) ?
            (this.state.projectStatusField.values.length === 0) : true;

        if (doesProjectHaveNoStatus) {
            fetchingProjectStatuses = true;
            this.fetchProjectStatuses();
        }

        this.setState({
            fetchingIssueMetadata,
            fetchingProjectStatuses,
            getMetaDataErr: null,
            filters,
        });
    };

    handleFilterFieldChange = (fields) => {
        this.setState({filters: {...this.state.filters, fields}});
        this.clearConflictingErrorMessage();
    };

    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        if (!this.validator.validate()) {
            return;
        }

        const filterFields = getCustomFieldFiltersForProjects(this.props.jiraIssueMetadata, this.state.filters.projects);
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
            name: this.state.subscriptionName,
        } as ChannelSubscription;

        this.setState({submitting: true, error: null});

        if (this.props.selectedSubscription) {
            subscription.id = this.props.selectedSubscription.id;
            this.props.editChannelSubscription(subscription).then((edited) => {
                if (edited.error) {
                    this.setState({error: edited.error.message, submitting: false});
                    return;
                }
                this.handleClose(e);
            });
        } else {
            this.props.createChannelSubscription(subscription).then((created) => {
                if (created.error) {
                    this.setState({error: created.error.message, submitting: false});
                    return;
                }
                this.handleClose(e);
            });
        }
    };

    render(): JSX.Element {
        const style = getStyle(this.props.theme);

        const projectOptions = getProjectValues(this.props.jiraProjectMetadata);
        const issueOptions = getIssueValuesForMultipleProjects(this.props.jiraProjectMetadata, this.state.filters.projects);
        const customFields = getCustomFieldValuesForEvents(this.props.jiraIssueMetadata, this.state.filters.projects);

        let filterFields = getCustomFieldFiltersForProjects(this.props.jiraIssueMetadata, this.state.filters.projects);
        if (this.state.projectStatusField && this.state.projectStatusField.values && this.state.projectStatusField.values.length !== 0) {
            // If user has no statuses in all jira project, then we will not show status as filter at all,
            // If there are then add it to filter field to create field for it
            filterFields = [...filterFields, this.state.projectStatusField];
        }

        const eventOptions = JiraEventOptions.concat(customFields);

        let conflictingErrorComponent = null;
        if (this.state.conflictingError) {
            conflictingErrorComponent = (
                <p className='help-text error-text'>
                    <span>{this.state.conflictingError}</span>
                </p>
            );
        }

        let component = null;
        if (this.props.channel && this.props.channelSubscriptions) {
            let innerComponent = null;

            // If its busy with fetching issue meta data or project status, show loading component instead
            if (this.state.fetchingIssueMetadata || this.state.fetchingProjectStatuses) {
                innerComponent = <Loading/>;
            } else if (this.state.filters.projects[0] && !this.state.getMetaDataErr &&
                this.props.jiraIssueMetadata && this.state.fetchingProjectStatuses === false) {
                innerComponent = (
                    <React.Fragment>
                        <ReactSelectSetting
                            name={'events'}
                            label={'Events'}
                            required={true}
                            onChange={this.handleSettingChange}
                            options={eventOptions}
                            isMulti={true}
                            theme={this.props.theme}
                            value={eventOptions.filter((option) => this.state.filters.events.includes(option.value))}
                            addValidate={this.validator.addComponent}
                            removeValidate={this.validator.removeComponent}
                        />
                        <ReactSelectSetting
                            name={'issue_types'}
                            label={'Issue Type'}
                            required={true}
                            onChange={this.handleIssueChange}
                            options={issueOptions}
                            isMulti={true}
                            theme={this.props.theme}
                            value={issueOptions.filter((option) => this.state.filters.issue_types.includes(option.value))}
                            addValidate={this.validator.addComponent}
                            removeValidate={this.validator.removeComponent}
                        />
                        {conflictingErrorComponent}
                        <ChannelSettingsFilters
                            fields={filterFields}
                            values={this.state.filters.fields}
                            chosenIssueTypes={this.state.filters.issue_types}
                            issueMetadata={this.props.jiraIssueMetadata}
                            theme={this.props.theme}
                            onChange={this.handleFilterFieldChange}
                            addValidate={this.validator.addComponent}
                            removeValidate={this.validator.removeComponent}
                        />
                        <div>
                            <label className='control-label margin-bottom'>
                                {'Approximate JQL Output'}
                            </label>
                            <div style={getBaseStyles(this.props.theme).codeBlock}>
                                <span>{generateJQLStringFromSubscriptionFilters(this.props.jiraIssueMetadata, filterFields, this.state.filters)}</span>
                            </div>
                        </div>
                    </React.Fragment>
                );
            }

            component = (
                <React.Fragment>
                    <div className='container-fluid'>
                        <Input
                            label={'Subscription Name'}
                            placeholder={'Name'}
                            type={'input'}
                            maxLength={100}
                            required={true}
                            onChange={this.handleNameChange}
                            value={this.state.subscriptionName}
                            readOnly={false}
                            addValidate={this.validator.addComponent}
                            removeValidate={this.validator.removeComponent}
                        />
                    </div>
                    <div className='container-fluid'>
                        <ReactSelectSetting
                            name={'projects'}
                            label={'Project'}
                            limitOptions={true}
                            required={true}
                            onChange={this.handleProjectChange}
                            options={projectOptions}
                            isMulti={false}
                            theme={this.props.theme}
                            value={projectOptions.filter((option) => this.state.filters.projects.includes(option.value))}
                            addValidate={this.validator.addComponent}
                            removeValidate={this.validator.removeComponent}
                        />
                        {innerComponent}
                    </div>
                </React.Fragment>
            );
        } else {
            component = <Loading/>;
        }

        const {showConfirmModal} = this.state;

        let confirmDeleteMessage = 'Delete Subscription?';
        if (this.props.selectedSubscription && this.props.selectedSubscription.name) {
            confirmDeleteMessage = `Delete Subscription "${this.props.selectedSubscription.name}"?`;
        }

        let confirmComponent;
        if (this.props.selectedSubscription) {
            confirmComponent = (
                <ConfirmModal
                    cancelButtonText={'Cancel'}
                    confirmButtonText={'Delete'}
                    confirmButtonClass={'btn btn-danger'}
                    hideCancel={false}
                    message={confirmDeleteMessage}
                    onCancel={this.handleCancelDelete}
                    onConfirm={this.handleConfirmDelete}
                    show={showConfirmModal}
                    title={'Subscription'}
                />
            );
        }

        let error = null;
        if (this.state.error || this.state.getMetaDataErr) {
            error = (
                <p className='help-text error-text'>
                    <span>{this.state.error || this.state.getMetaDataErr}</span>
                </p>
            );
        }

        const enableSubmitButton = Boolean(this.state.filters.projects[0]);
        const enableDeleteButton = Boolean(this.props.selectedSubscription);

        let saveSubscriptionButtonText = 'Save Subscription';
        let headerText = 'Edit Jira Subscription for ';
        if (this.props.creatingSubscription) {
            saveSubscriptionButtonText = 'Add Subscription';
            headerText = 'Add Jira Subscription in ';
        }

        return (
            <form
                role='form'
            >
                <div className='margin-bottom x3 text-center'>
                    <h2>{headerText}<strong>{this.props.channel.display_name}</strong></h2>
                </div>
                <div style={style.modalBody}>
                    {component}
                    {error}
                    {confirmComponent}
                </div>
                <Modal.Footer style={style.modalFooter}>
                    <FormButton
                        id='jira-delete-subscription'
                        type='button'
                        btnClass='btn-danger pull-left'
                        defaultMessage='Delete'
                        disabled={!enableDeleteButton}
                        onClick={this.handleDeleteChannelSubscription}
                    />
                    <FormButton
                        type='button'
                        btnClass='btn-link'
                        defaultMessage='Cancel'
                        onClick={this.handleClose}
                    />
                    <FormButton
                        type='button'
                        onClick={this.handleCreate}
                        disabled={!enableSubmitButton}
                        btnClass='btn-primary'
                        saving={this.state.submitting}
                        defaultMessage={saveSubscriptionButtonText}
                        savingMessage='Saving...'
                    />
                </Modal.Footer>
            </form>
        );
    }
}

const getStyle = (theme: any): any => ({
    modalBody: {
        padding: '2em 0',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
    modalFooter: {
        padding: '2rem 15px',
    },
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000',
    },
});
