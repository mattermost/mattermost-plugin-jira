// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

export default class Loading extends PureComponent {
    static propTypes = {
        position: PropTypes.oneOf(['absolute', 'fixed', 'relative', 'static', 'inherit']),
        style: PropTypes.object,
    };

    static defaultProps = {
        position: 'relative',
        style: {},
    };

    render() {
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
