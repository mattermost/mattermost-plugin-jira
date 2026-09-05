// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {RHSSort} from 'types/rhs';

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
    const onSortChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
        const next = event.target.value;
        if (next === 'updated' || next === 'created') {
            props.onSortChange(next);
        }
    };

    return (
        <div
            className='jira-rhs-header'
            data-testid='rhs-header'
        >
            {props.instancePicker}
            <div className='jira-rhs-header-actions'>
                <select
                    className='jira-rhs-select'
                    aria-label={RHS_STRINGS.sortLabel}
                    data-testid='rhs-sort'
                    value={props.sort}
                    onChange={onSortChange}
                >
                    {SORT_OPTIONS.map((option) => (
                        <option
                            key={option.value}
                            value={option.value}
                        >
                            {option.label}
                        </option>
                    ))}
                </select>
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
