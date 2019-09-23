// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import Loading from 'components/loading';

import FullScreenModal from '../full_screen_modal/full_screen_modal';

import ChannelSettingsModalInner from './channel_settings_internal';

import './channel_settings_modal.scss';

export default class ChannelSettingsModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        channel: PropTypes.object,
        channelSubscriptions: PropTypes.array,
        jiraProjectMetadata: PropTypes.object,
        fetchJiraProjectMetadata: PropTypes.func.isRequired,
        fetchChannelSubscriptions: PropTypes.func.isRequired,
    };

    componentDidUpdate(prevProps) {
        if (this.props.channel && (!prevProps.channel || this.props.channel.id !== prevProps.channel.id)) {
            this.props.fetchJiraProjectMetadata();
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
        if (this.props.channelSubscriptions && this.props.jiraProjectMetadata) {
            if (this.props.channelSubscriptions instanceof Error) {
                inner = (
                    <Modal.Body>
                        {'You do not have permission to access Jira subscriptions in this channel.'}
                    </Modal.Body>
                );
            } else {
                inner = (
                    <ChannelSettingsModalInner
                        {...this.props}
                    />
                );
            }
        }

        return (
            <FullScreenModal
                show={Boolean(this.props.channel)}
                onClose={this.handleClose}
            >
                <div className='channel-subscriptions-modal'>
                    <div className='channel-subscriptions-modal-body'>
                        {inner}
                    </div>
                </div>
            </FullScreenModal>
        );
    }
}
