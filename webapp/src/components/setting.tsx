// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ErrorMessage} from '@mattermost/compass-ui/components/error-message';

import './form.scss';

export const REQUIRED_MESSAGE = 'This field is required.';

export function fieldLabel(label: React.ReactNode, required?: boolean): React.ReactNode {
    if (!required) {
        return label;
    }

    return (
        <>
            {label}
            <span
                className='jira-setting__required'
                aria-hidden={true}
            >
                {' *'}
            </span>
        </>
    );
}

type Props = {
    children: React.ReactNode;
    error?: React.ReactNode;
};

export default function Setting({children, error}: Props) {
    return (
        <div className='jira-setting'>
            {children}
            {error && <ErrorMessage message={error}/>}
        </div>
    );
}
