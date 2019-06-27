// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import Loading from 'components/loading';

import ChannelSettingsModalInner from './channel_settings_internal.jsx';

export default class ChannelSettingsModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        channel: PropTypes.object,
        channelSubscriptions: PropTypes.Array,
        jiraIssueMetadata: PropTypes.object,
        fetchJiraIssueMetadata: PropTypes.func.isRequired,
        fetchChannelSubscriptions: PropTypes.func.isRequired,
    }
    componentDidUpdate(prevProps) {
        if (this.props.channel && (!prevProps.channel || this.props.channel.id !== prevProps.channel.id)) {
            this.props.fetchJiraIssueMetadata();
            this.props.fetchChannelSubscriptions(this.props.channel.id);
        }
    }
    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        this.props.close();
    };
    render() {
        let inner = <Loading/>;
        if (this.props.channelSubscriptions && this.props.jiraIssueMetadata) {
            inner = (
                <ChannelSettingsModalInner
                    {...this.props}
                />
            );
        }

        return (
            <Modal
                dialogClassName='modal--scroll'
                show={Boolean(this.props.channel)}
                onHide={this.handleClose}
                onExited={this.handleClose}
                bsSize='large'
            >
                <Modal.Header closeButton={true}>
                    <Modal.Title>
                        {'Channel Jira Settings'}
                    </Modal.Title>
                </Modal.Header>
                {inner}
            </Modal>
        );
    }
}
