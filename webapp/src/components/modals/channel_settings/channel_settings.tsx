// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {ProjectMetadata, ChannelSubscription} from 'types/model';

import FullScreenModal from '../full_screen_modal/full_screen_modal';

import ChannelSettingsModalInner from './channel_settings_internal';
import {SharedProps} from './shared_props';

import './channel_settings_modal.scss';

export type Props = SharedProps;

export default class ChannelSettingsModal extends PureComponent<Props> {
    handleClose = (e?: Event): void => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.close();
    };

    render(): JSX.Element {
        const isModalOpen = Boolean(this.props.channel);

        let inner;
        if (isModalOpen) {
            inner = (
                <ChannelSettingsModalInner
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
