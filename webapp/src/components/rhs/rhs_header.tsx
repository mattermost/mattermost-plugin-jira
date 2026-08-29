// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {RHSSort} from 'types/model';

import RHSSelect from './rhs_select';
import {RHS_STRINGS} from './rhs_strings';

export type Props = {
    sort: RHSSort;
    loading: boolean;
    onSortChange: (sort: RHSSort) => void;
    onRefresh: () => void;
    onNewTicket: () => void;
    instancePicker?: React.ReactNode;
};

const SORT_OPTIONS: Array<{value: RHSSort; label: string}> = [
    {value: 'updated', label: RHS_STRINGS.sortUpdated},
    {value: 'created', label: RHS_STRINGS.sortCreated},
];

export default function RHSHeader(props: Props): JSX.Element {
    return (
        <div
            className='jira-rhs-header'
            data-testid='rhs-header'
        >
            {props.instancePicker}
            <div className='jira-rhs-header-actions'>
                <RHSSelect
                    ariaLabel={RHS_STRINGS.sortLabel}
                    testId='rhs-sort'
                    value={props.sort}
                    options={SORT_OPTIONS}
                    onChange={props.onSortChange}
                />
                <button
                    type='button'
                    className='btn btn-icon btn-sm'
                    aria-label={RHS_STRINGS.refresh}
                    data-testid='rhs-refresh'
                    disabled={props.loading}
                    onClick={props.onRefresh}
                >
                    <i
                        className='icon icon-refresh'
                        aria-hidden={true}
                    />
                </button>
                <button
                    type='button'
                    className='btn btn-primary btn-sm jira-rhs-new-ticket'
                    data-testid='rhs-new-ticket'
                    onClick={props.onNewTicket}
                >
                    <i
                        className='icon icon-plus'
                        aria-hidden={true}
                    />
                    <span>{RHS_STRINGS.newTicket}</span>
                </button>
            </div>
        </div>
    );
}
