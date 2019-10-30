// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {Channel} from 'types/model';

import JiraIcon from 'components/icon';

type Registry = {
    registerChannelHeaderButtonAction: (
        icon: React.ReactNode,
        cb: (channel: Channel) => void,
        name: string
    ) => string;
    unregisterComponent: (id: string) => void;
};

type Props = {
    userConnected?: boolean;
    instanceInstalled?: boolean;
    registry: Registry;
    openChannelSettings: (channelID: string) => void;
    haveSetupUI: boolean;
    finishedSetupUI: () => void;
    setupUI: () => void;
    setHeaderButtonId: (id: string) => void;
    headerButtonId: string;
}

// SetupUI is a dummy Root component that we use to detect when the user has logged in
export default class SetupUI extends PureComponent<Props> {
    registerHeaderButton = (): void => {
        const id = this.props.registry.registerChannelHeaderButtonAction(
            <JiraIcon/>,
            (channel: Channel) => this.props.openChannelSettings(channel.id),
            'Jira',
        );
        this.props.setHeaderButtonId(id);
    };

    componentDidMount(): void {
        if (this.props.instanceInstalled && this.props.userConnected && this.props.headerButtonId === '') {
            this.registerHeaderButton();
        }
        if (!this.props.haveSetupUI) {
            this.props.setupUI();
            this.props.finishedSetupUI();
        }
    }

    componentDidUpdate(): void {
        if (!this.props.userConnected || !this.props.instanceInstalled) {
            if (this.props.headerButtonId !== '') {
                this.props.registry.unregisterComponent(this.props.headerButtonId);
                this.props.setHeaderButtonId('');
            }
        }

        if (this.props.userConnected && this.props.instanceInstalled) {
            if (this.props.headerButtonId === '') {
                this.registerHeaderButton();
            }
        }
    }

    render(): null {
        return null;
    }
}
