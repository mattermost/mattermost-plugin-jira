// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {Instance} from 'types/model';

import {RHS_STRINGS} from './rhs_strings';

export type Props = {
    instances: Instance[];
    selectedInstanceID: string;
    onChange: (instanceID: string) => void;
};

export default function RHSInstancePicker(props: Props): JSX.Element | null {
    if (props.instances.length <= 1) {
        return null;
    }

    return (
        <select
            className='jira-rhs-select'
            aria-label={RHS_STRINGS.instanceLabel}
            data-testid='rhs-instance-picker'
            value={props.selectedInstanceID}
            onChange={(event) => props.onChange(event.target.value)}
        >
            {props.instances.map((instance) => (
                <option
                    key={instance.instance_id}
                    value={instance.instance_id}
                >
                    {instance.alias || instance.instance_id}
                </option>
            ))}
        </select>
    );
}
