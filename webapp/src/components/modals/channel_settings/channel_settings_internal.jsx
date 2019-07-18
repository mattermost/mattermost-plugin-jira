// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import ReactSelectSetting from 'components/react_select_setting';
import FormButton from 'components/form_button';
import Loading from 'components/loading';
import {getProjectValues, getIssueValuesForMultipleProjects, getCustomFieldValuesForProject} from 'utils/jira_issue_metadata';

const JiraEventOptions = [
    {value: 'event_created', label: 'Issue Created'},
    {value: 'event_deleted', label: 'Issue Deleted'},
    {value: 'event_deleted_unresolved', label: 'Issue Deleted, Unresolved'},
    {value: 'event_updated_reopened', label: 'Issue Reopened'},
    {value: 'event_updated_resolved', label: 'Issue Resolved'},
    {value: 'event_created_comment', label: 'Comment Created'},
    {value: 'event_updated_comment', label: 'Comment Updated'},
    {value: 'event_deleted_comment', label: 'Comment Deleted'},
    {value: 'event_updated_any', label: 'Issue Updated: Any'},
    {value: 'event_updated_assignee', label: 'Issue Updated: Assignee'},
    {value: 'event_updated_attachment', label: 'Issue Updated: Attachment'},
    {value: 'event_updated_description', label: 'Issue Updated: Description'},
    {value: 'event_updated_fix_version', label: 'Issue Updated: Fix Version'},
    {value: 'event_updated_issue_type', label: 'Issue Updated: Issue Type'},
    {value: 'event_updated_labels', label: 'Issue Updated: Labels'},
    {value: 'event_updated_priority', label: 'Issue Updated: Priority'},
    {value: 'event_updated_rank', label: 'Issue Updated: Rank'},
    {value: 'event_updated_sprint', label: 'Issue Updated: Sprint'},
    {value: 'event_updated_status', label: 'Issue Updated: Status'},
    {value: 'event_updated_summary', label: 'Issue Updated: Summary'},
];

export default class ChannelSettingsModalInner extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        channel: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
        jiraProjectMetadata: PropTypes.object.isRequired,
        jiraIssueMetadata: PropTypes.object.isRequired,
        channelSubscriptions: PropTypes.array.isRequired,
        createChannelSubscription: PropTypes.func.isRequired,
        deleteChannelSubscription: PropTypes.func.isRequired,
        editChannelSubscription: PropTypes.func.isRequired,
        fetchChannelSubscriptions: PropTypes.func.isRequired,
        fetchJiraIssueMetadataForProject: PropTypes.func.isRequired,
        clearIssueMetadata: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);

        let filters = {
            events: [],
            projects: [],
            issue_types: [],
        };

        if (props.channelSubscriptions[0]) {
            filters = Object.assign({}, filters, props.channelSubscriptions[0].filters);
        }

        let fetchingProject = false;
        if (filters.projects.length) {
            fetchingProject = true;
            this.fetchProject(filters.projects[0]);
        }

        this.state = {
            error: null,
            getMetaDataErr: null,
            submitting: false,
            filters,
            fetchingProject,
        };
    }

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        this.props.close();
    };

    deleteChannelSubscription = (e) => {
        if (this.props.channelSubscriptions && this.props.channelSubscriptions.length > 0) {
            const sub = this.props.channelSubscriptions[0];
            this.props.deleteChannelSubscription(sub);
        }
        this.handleClose(e);
    }

    handleSettingChange = (id, value) => {
        let finalValue = value;
        if (!Array.isArray(finalValue)) {
            finalValue = [finalValue];
        }
        const filters = {...this.state.filters};
        filters[id] = finalValue;
        this.setState({filters});
    };

    fetchProject = (projectKey) => {
        this.props.fetchJiraIssueMetadataForProject(projectKey).then((fetched) => {
            const state = {fetchingProject: false};
            if (fetched.error) {
                state.getMetaDataErr = fetched.error.message;
            }
            this.setState(state);
        });
    };

    handleProjectChange = (id, value) => {
        const projectKey = value;

        const filters = {
            projects: [value],
            events: [],
            issue_types: [],
        };
        this.setState({
            fetchingProject: true,
            getMetaDataErr: null,
            filters,
        });

        this.props.clearIssueMetadata();
        this.fetchProject(projectKey);
    };

    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        const projects = this.state.filters.projects;
        if (!projects.length) {
            this.setState({error: 'Please select a project.'});
            return;
        }

        const events = this.state.filters.events;
        if (!events.length) {
            this.setState({error: 'Please select an event type.'});
            return;
        }

        const issueTypes = this.state.filters.issue_types;
        if (!issueTypes.length) {
            this.setState({error: 'Please select an issue type.'});
            return;
        }

        const subscription = {
            channel_id: this.props.channel.id,
            filters: this.state.filters,
        };

        this.setState({submitting: true, error: null});

        if (this.props.channelSubscriptions && this.props.channelSubscriptions.length > 0) {
            subscription.id = this.props.channelSubscriptions[0].id;
            this.props.editChannelSubscription(subscription).then((edited) => {
                if (edited.error) {
                    this.setState({error: edited.error.message, submitting: false});
                    return;
                }
                this.props.fetchChannelSubscriptions(this.props.channel.id);
                this.handleClose(e);
            });
        } else {
            this.props.createChannelSubscription(subscription).then((created) => {
                if (created.error) {
                    this.setState({error: created.error.message, submitting: false});
                    return;
                }
                this.props.fetchChannelSubscriptions(this.props.channel.id);
                this.handleClose(e);
            });
        }
    };

    render() {
        const style = getStyle(this.props.theme);

        const projectOptions = getProjectValues(this.props.jiraProjectMetadata);
        const issueOptions = getIssueValuesForMultipleProjects(this.props.jiraProjectMetadata, this.state.filters.projects);
        const customFields = getCustomFieldValuesForProject(this.props.jiraIssueMetadata, this.state.filters.projects[0]);

        const eventOptions = JiraEventOptions.concat(customFields);

        let component = null;
        if (this.props.channel && this.props.channelSubscriptions) {
            let innerComponent = null;
            if (this.state.fetchingProject) {
                innerComponent = <Loading/>;
            } else if (this.state.filters.projects[0] && !this.state.getMetaDataErr) {
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
                        />
                        <ReactSelectSetting
                            name={'issue_types'}
                            label={'Issue Type'}
                            required={true}
                            onChange={this.handleSettingChange}
                            options={issueOptions}
                            isMulti={true}
                            theme={this.props.theme}
                            value={issueOptions.filter((option) => this.state.filters.issue_types.includes(option.value))}
                        />
                    </React.Fragment>
                );
            }

            component = (
                <div style={style.modal}>
                    <ReactSelectSetting
                        name={'projects'}
                        label={'Project'}
                        required={true}
                        onChange={this.handleProjectChange}
                        options={projectOptions}
                        isMulti={false}
                        theme={this.props.theme}
                        value={projectOptions.filter((option) => this.state.filters.projects.includes(option.value))}
                    />
                    {innerComponent}
                    <br/>
                </div>
            );
        } else {
            component = <Loading/>;
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
        const showDeleteButton = Boolean(this.props.channelSubscriptions && this.props.channelSubscriptions.length > 0);

        return (
            <form
                role='form'
                onSubmit={this.handleCreate}
            >
                <Modal.Body ref='modalBody'>
                    {component}
                    {error}
                </Modal.Body>
                <Modal.Footer>
                    <FormButton
                        type='button'
                        btnClass='btn-link'
                        defaultMessage='Cancel'
                        onClick={this.handleClose}
                    />
                    {showDeleteButton && (
                        <FormButton
                            type='button'
                            btnClass='btn-danger'
                            defaultMessage='Delete'
                            onClick={this.deleteChannelSubscription}
                        />
                    )}
                    <FormButton
                        type='submit'
                        disabled={!enableSubmitButton}
                        btnClass='btn-primary'
                        saving={this.state.submitting}
                        defaultMessage='Set Subscription'
                        savingMessage='Setting'
                    />
                </Modal.Footer>
            </form>
        );
    }
}

const getStyle = (theme) => ({
    modal: {
        padding: '1em',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000',
    },
});
