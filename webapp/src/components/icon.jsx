// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import PropTypes from 'prop-types';
import React from 'react';

export default class ChannelHeaderButtonIcon extends React.PureComponent {
    static propTypes = {
        type: PropTypes.string,
    }

    render() {
        let iconStyle = {};
        if (this.props.type === 'menu') {
            iconStyle = {width: '20px', height: '20px', fill: '#0052CC', 'margin-right': '8px', background: 'white', 'border-radius': '50px', padding: '2px'};
        }

        return (
            <svg
                aria-hidden='true'
                focusable='false'
                role='img'
                viewBox='0 0 496 512'
                width='14'
                height='14'
                style={iconStyle}
            >
                <path d='M490 241.7C417.1 169 320.6 71.8 248.5 0 83 164.9 6 241.7 6 241.7c-7.9 7.9-7.9 20.7 0 28.7C138.8 402.7 67.8 331.9 248.5 512c379.4-378 15.7-16.7 241.5-241.7 8-7.9 8-20.7 0-28.6zm-241.5 90l-76-75.7 76-75.7 76 75.7-76 75.7z'/>
            </svg>
        );
    }
}
