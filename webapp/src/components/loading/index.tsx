// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import './loading.scss';

const Loading = () => {
    return (
        <div
            className='loading-screen'
            style={{position: 'relative'}}
        >
            <div className='loading__content'>
                <h3 className='loading-text'>{'Loading'}</h3>
                <div className='round round-1'/>
                <div className='round round-2'/>
                <div className='round round-3'/>
            </div>
        </div>
    );
};

export default Loading;
