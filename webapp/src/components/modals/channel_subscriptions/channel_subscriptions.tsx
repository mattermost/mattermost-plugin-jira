// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {ModalOverlay} from '@mattermost/compass-ui/components/modal';

import {AllProjectMetadata} from 'types/model';

import {Z_INDEX_MODAL} from 'utils/styles';

import ChannelSubscriptionsModalInner from './channel_subscriptions_internal';
import {SharedProps} from './shared_props';

import './channel_subscriptions_modal.scss';

export type Props = SharedProps;

type State = {
    showModal: boolean;
    allProjectMetadata: AllProjectMetadata | null;
};

export default class ChannelSubscriptionsModal extends PureComponent<Props, State> {
    state = {
        showModal: false,
        allProjectMetadata: null,
    };

    private inner = React.createRef<ChannelSubscriptionsModalInner>();

    // Keeps the last open view rendered while the overlay plays its exit.
    private lastOpenProps: Props | null = null;
    private session = 0;
    private wasOpen = false;

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

        const templatesResponse = await this.props.fetchAllSubscriptionTemplates(this.props.channel.id);

        this.setState({showModal: true, allProjectMetadata: projectResponses.data});
    };

    handleOverlayClose = () => {
        if (this.inner.current) {
            this.inner.current.requestClose();
        } else {
            this.props.close();
        }
    };

    render(): JSX.Element {
        const isModalOpen = Boolean(this.props.channel && this.state.showModal);
        if (isModalOpen) {
            if (!this.wasOpen) {
                this.session += 1;
            }
            this.lastOpenProps = this.props;
        }
        this.wasOpen = isModalOpen;

        let inner;
        if (this.lastOpenProps) {
            inner = (
                <ChannelSubscriptionsModalInner
                    key={this.session}
                    ref={this.inner}
                    allProjectMetadata={this.state.allProjectMetadata}
                    {...this.lastOpenProps}
                    open={isModalOpen}
                />
            );
        }

        return (
            <ModalOverlay
                open={isModalOpen}
                onClose={this.handleOverlayClose}
                onExited={() => {
                    this.lastOpenProps = null;
                }}
                zIndex={Z_INDEX_MODAL}
            >
                {inner}
            </ModalOverlay>
        );
    }
}
