// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {PureComponent} from 'react';

type Props = {
    haveSetupUI: boolean;
    finishedSetupUI: () => void;
    setupUI: () => void;
};

// SetupUI is a dummy Root component that we use to detect when the user has logged in
export default class SetupUI extends PureComponent<Props> {
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
