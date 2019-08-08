// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PureComponent} from 'react';
import PropTypes from 'prop-types';

// SetupUI is a dummy Root component that we use to detect when the user has logged in
export default class SetupUI extends PureComponent {
    static propTypes = {
        haveSetupUI: PropTypes.bool.isRequired,
        finishedSetupUI: PropTypes.func.isRequired,
        setupUI: PropTypes.func.isRequired,
    };

    componentDidMount() {
        if (!this.props.haveSetupUI) {
            this.props.setupUI();
            this.props.finishedSetupUI();
        }
    }

    render() {
        return null;
    }
}
