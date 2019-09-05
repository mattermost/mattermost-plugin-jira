// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

type Props = {
    position?: 'absolute' | 'fixed' | 'relative' | 'static' | 'inherit';
    style?: object;
};

export default class Loading extends PureComponent<Props> {
    static defaultProps = {
        position: 'relative',
        style: {},
    };

    public render() {
        return (
            <div
                className='loading-screen'
                style={{position: this.props.position, ...this.props.style}}
            >
                <div className='loading__content'>
                    <h3>
                        {'Loading'}
                    </h3>
                    <div className='round round-1'/>
                    <div className='round round-2'/>
                    <div className='round round-3'/>
                </div>
            </div>
        );
    }
}
