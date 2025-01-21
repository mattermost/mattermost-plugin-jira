// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import './defaultAvatar.scss';

function DefaultAvatar() {
    return (
        <span className='default-avatar'>
            <svg
                width='18'
                height='18'
                viewBox='0 0 18 18'
                role='presentation'
            >
                <g
                    fill='white'
                    fillRule='evenodd'
                >
                    <path
                        d='M3.5 14c0-1.105.902-2 2.009-2h7.982c1.11 0 2.009.894 2.009 2.006v4.44c0 3.405-12 3.405-12 0V14z'
                    />
                    <circle
                        cx='9'
                        cy='6'
                        r='3.5'
                    />
                </g>
            </svg>
        </span>
    );
}

export default DefaultAvatar;
