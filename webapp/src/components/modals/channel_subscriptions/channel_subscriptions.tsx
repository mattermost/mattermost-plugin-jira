// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import FullScreenModal from 'components/modals/full_screen_modal/full_screen_modal';

import {AllProjectMetadata} from 'types/model';

import ChannelSubscriptionsModalInner from './channel_subscriptions_internal';
import {SharedProps} from './shared_props';

import './channel_subscriptions_modal.scss';

export type Props = SharedProps;

type State = {
    showModal: boolean;
    allProjectMetadata: AllProjectMetadata | null
};

export default class ChannelSubscriptionsModal extends PureComponent<Props, State> {
    state = {
        showModal: false,
        allProjectMetadata: null,
    };

    componentDidUpdate(prevProps: Props) {
        if (prevProps.channel && !this.props.channel) {
            this.handleModalClosed();
        } else if (!prevProps.channel && this.props.channel) {
            this.handleModalOpened();
        }
    }

    handleModalClosed = () => {
        this.setState({showModal: false});
    };

    handleModalOpened = () => {
        this.fetchData();
    };

    fetchData = async (): Promise<void> => {
        if (!this.props.channel) {
            return;
        }

        const subsResponse = await this.props.fetchChannelSubscriptions(this.props.channel.id);
        if (subsResponse.error) {
            this.props.sendEphemeralPost('You do not have permission to edit subscriptions for this channel. Subscribing to Jira events will create notifications in this channel when certain events occur, such as an issue being updated or created with a specific label. Speak to your Mattermost administrator to request access to this functionality.');
            this.props.close();
            return;
        }

        const projectResponses = await this.props.fetchJiraProjectMetadataForAllInstances();
        if (projectResponses.error) {
            this.props.sendEphemeralPost('Failed to fetch project metadata for any projects.');
            this.props.close();
            return;
        }

        this.setState({showModal: true, allProjectMetadata: projectResponses.data});
    };

    handleClose = (e?: Event): void => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.close();
    };

    render(): JSX.Element {
        const isModalOpen = Boolean(this.props.channel && this.state.showModal);

        let inner;
        if (isModalOpen) {
            inner = (
                <ChannelSubscriptionsModalInner
                    allProjectMetadata={this.state.allProjectMetadata}
                    {...this.props}
                />
            );
        }

        return (
            <FullScreenModal
                show={isModalOpen}
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
