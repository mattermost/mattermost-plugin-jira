// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PureComponent} from 'react';

import {setupUI} from 'plugin';

// SetupUI is a dummy Root component that we use to detect when the user has logged in
export default class SetupUI extends PureComponent {
    componentDidMount() {
        setupUI();
    }

    render() {
        return null;
    }
}
