// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PureComponent} from 'react';
import PropTypes from 'prop-types';

import JiraIcon from 'components/icon';

// SetupUI is a dummy Root component that we use to detect when the user has logged in
export default class SetupUI extends PureComponent {
    static propTypes = {
        userConnected: PropTypes.bool,
        instanceInstalled: PropTypes.bool,
        registry: PropTypes.object.isRequired,
        openChannelSettings: PropTypes.func.isRequired,
        haveSetupUI: PropTypes.bool.isRequired,
        finishedSetupUI: PropTypes.func.isRequired,
        setupUI: PropTypes.func.isRequired,
        setHeaderButtonId: PropTypes.func.isRequired,
        headerButtonId: PropTypes.string.isRequired,
    };

    registerHeaderButton = () => {
        const id = this.props.registry.registerChannelHeaderButtonAction(
            <JiraIcon/>,
            (channel) => this.props.openChannelSettings(channel.id),
            'Jira',
        );
        this.props.setHeaderButtonId(id);
    };

    componentDidMount() {
        if (this.props.instanceInstalled && this.props.userConnected && this.props.headerButtonId === '') {
            this.registerHeaderButton();
        }
        if (!this.props.haveSetupUI) {
            this.props.setupUI();
            this.props.finishedSetupUI();
        }
    }

    componentDidUpdate() {
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

    render() {
        return null;
    }
}
