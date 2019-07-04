// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PureComponent} from 'react';
import PropTypes from 'prop-types';

import {setupUI} from 'plugin';

import JiraIcon from 'components/icon';

// SetupUI is a dummy Root component that we use to detect when the user has logged in
export default class SetupUI extends PureComponent {
    static propTypes = {
        userConnected: PropTypes.bool.isRequired,
        instanceInstalled: PropTypes.bool.isRequired,
        registry: PropTypes.object.isRequired,
        openChannelSettings: PropTypes.func.isRequired,
    };

    registerHeaderButton = () => {
        this.headerButtonId = this.props.registry.registerChannelHeaderButtonAction(
            <JiraIcon/>,
            (channel) => this.props.openChannelSettings(channel.id),
            'JIRA',
        );
    }

    componentDidMount() {
        if (this.props.instanceInstalled && this.props.userConnected) {
            this.registerHeaderButton();
        }
        setupUI();
    }

    componentDidUpdate() {
        if (!this.props.userConnected || !this.props.instanceInstalled) {
            if (this.headerButtonId) {
                this.props.registry.unregisterComponent(this.headerButtonId);
                this.headerButtonId = null;
            }
        }

        if (this.props.userConnected && this.props.instanceInstalled) {
            if (!this.headerButtonId) {
                this.registerHeaderButton();
            }
        }
    }

    render() {
        return null;
    }
}
