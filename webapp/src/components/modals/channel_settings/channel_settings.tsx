// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Modal} from 'react-bootstrap';

import Loading from 'components/loading';
import {ProjectMetadata, ChannelSubscription} from 'types/model';

import FullScreenModal from '../full_screen_modal/full_screen_modal';

import ChannelSettingsModalInner from './channel_settings_internal';
import {SharedProps} from './shared_props';

import './channel_settings_modal.scss';

type Props = SharedProps & {
    fetchJiraProjectMetadata: () => Promise<{data: ProjectMetadata}>;
    fetchChannelSubscriptions: (channelId: string) => Promise<{data: ChannelSubscription[]}>;
    close: () => void;
}

export default class ChannelSettingsModal extends PureComponent<Props> {
    componentDidUpdate(prevProps: Props): void {
        if (this.props.channel && (!prevProps.channel || this.props.channel.id !== prevProps.channel.id)) {
            this.props.fetchJiraProjectMetadata();
            this.props.fetchChannelSubscriptions(this.props.channel.id);
        }
    }

    handleClose = (e: Event): void => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        this.props.close();
    };

    render(): JSX.Element {
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
