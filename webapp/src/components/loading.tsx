// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {Spinner} from '@mattermost/compass-ui/components/spinner';

import './form.scss';

const Loading = () => {
    return (
        <div className='jira-loading'>
            <Spinner size='32'/>
        </div>
    );
};

export default Loading;
