// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import ReactSelectSetting from 'components/react_select_setting';
import FormButton from 'components/form_button';
import Loading from 'components/loading';
import {getProjectValues, getIssueValuesForMultipleProjects} from 'utils/jira_issue_metadata';

const JiraEventOptions = [
    {value: 'jira:issue_created', label: 'Issue Created'},
    {value: 'jira:issue_updated', label: 'Issue Updated'},
    {value: 'jira:issue_deleted', label: 'Issue Deleted'},
];

export default class ChannelSettingsModalInner extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        channel: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
        jiraIssueMetadata: PropTypes.object.isRequired,
        channelSubscriptions: PropTypes.array.isRequired,
        createChannelSubscription: PropTypes.func.isRequired,
        deleteChannelSubscription: PropTypes.func.isRequired,
        editChannelSubscription: PropTypes.func.isRequired,
        fetchChannelSubscriptions: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);

        let filters = {
            events: [],
            project: [],
            issue_type: [],
        };

        if (props.channelSubscriptions[0]) {
            filters = Object.assign({}, filters, props.channelSubscriptions[0].filters);
        }

        this.state = {
            error: null,
            submitting: false,
            filters,
        };
    }

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        this.props.close();
    };

    handleSettingChange = (id, value) => {
        let finalValue = value;
        if (!Array.isArray(finalValue)) {
            finalValue = [finalValue];
        }
        const filters = {...this.state.filters};
        filters[id] = finalValue;
        this.setState({filters});
    };

    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        const subscription = {
            channel_id: this.props.channel.id,
            filters: this.state.filters,
        };

        this.setState({submitting: true});

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
        const projectOptions = getProjectValues(this.props.jiraIssueMetadata);
        const issueOptions = getIssueValuesForMultipleProjects(this.props.jiraIssueMetadata, this.state.filters.project);

        let component = null;
        if (this.props.channel && this.props.channelSubscriptions) {
            component = (
                <div style={style.modal}>
                    <ReactSelectSetting
                        name={'events'}
                        label={'Events'}
                        required={true}
                        onChange={this.handleSettingChange}
                        options={JiraEventOptions}
                        isMulti={true}
                        theme={this.props.theme}
                        value={JiraEventOptions.filter((option) => this.state.filters.events.includes(option.value))}
                    />
                    <ReactSelectSetting
                        name={'project'}
                        label={'Project'}
                        required={false}
                        onChange={this.handleSettingChange}
                        options={projectOptions}
                        isMulti={true}
                        theme={this.props.theme}
                        value={projectOptions.filter((option) => this.state.filters.project.includes(option.value))}
                    />
                    <ReactSelectSetting
                        name={'issue_type'}
                        label={'Issue Type'}
                        required={false}
                        onChange={this.handleSettingChange}
                        options={issueOptions}
                        isMulti={true}
                        theme={this.props.theme}
                        value={issueOptions.filter((option) => this.state.filters.issue_type.includes(option.value))}
                    />
                    <br/>
                </div>
            );
        } else {
            component = <Loading/>;
        }

        return (
            <form
                role='form'
                onSubmit={this.handleCreate}
            >
                <Modal.Body ref='modalBody'>
                    {component}
                </Modal.Body>
                <Modal.Footer>
                    <FormButton
                        type='button'
                        btnClass='btn-default'
                        defaultMessage='Cancel'
                        onClick={this.handleClose}
                    />
                    <FormButton
                        type='submit'
                        btnClass='btn btn-primary'
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
