// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {Instance} from 'types/model';

import RHSSelect from './rhs_select';
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
        <RHSSelect
            ariaLabel={RHS_STRINGS.instanceLabel}
            testId='rhs-instance-picker'
            value={props.selectedInstanceID}
            options={props.instances.map((instance) => {
                return {
                    value: instance.instance_id,
                    label: instance.alias || instance.instance_id,
                };
            })}
            onChange={props.onChange}
        />
    );
}
