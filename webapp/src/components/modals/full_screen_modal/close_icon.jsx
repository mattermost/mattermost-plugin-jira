// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

export default class CloseIcon extends React.PureComponent {
    render() {
        return (
            <span {...this.props}>
                <svg
                    width='24px'
                    height='24px'
                    viewBox='0 0 24 24'
                    role='icon'
                >
                    <path d='M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z'/>
                </svg>
            </span>
        );
    }
}
