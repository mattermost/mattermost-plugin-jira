// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {RHSSort} from 'types/model';

import {RHS_STRINGS} from './rhs_strings';

export type Props = {
    sort: RHSSort;
    loading: boolean;
    onSortChange: (sort: RHSSort) => void;
    onRefresh: () => void;
    onNewTicket: () => void;
    instancePicker?: React.ReactNode;
};

export default function RHSHeader(props: Props): JSX.Element {
    return (
        <div
            className='jira-rhs-header'
            data-testid='rhs-header'
        >
            {props.instancePicker}
            <div className='jira-rhs-header-actions'>
                <label className='jira-rhs-sort'>
                    <span className='jira-rhs-sort-label'>{RHS_STRINGS.sortLabel}</span>
                    <select
                        aria-label={RHS_STRINGS.sortLabel}
                        data-testid='rhs-sort'
                        value={props.sort}
                        onChange={(event) => props.onSortChange(event.target.value as RHSSort)}
                    >
                        <option value='updated'>{RHS_STRINGS.sortUpdated}</option>
                        <option value='created'>{RHS_STRINGS.sortCreated}</option>
                    </select>
                </label>
                <button
                    type='button'
                    className='jira-rhs-icon-button'
                    aria-label={RHS_STRINGS.refresh}
                    data-testid='rhs-refresh'
                    disabled={props.loading}
                    onClick={props.onRefresh}
                >
                    <i
                        className='fa fa-refresh'
                        title={RHS_STRINGS.refresh}
                    />
                </button>
                <button
                    type='button'
                    className='btn btn-primary jira-rhs-new-ticket'
                    data-testid='rhs-new-ticket'
                    onClick={props.onNewTicket}
                >
                    <i className='fa fa-plus'/>
                    <span>{RHS_STRINGS.newTicket}</span>
                </button>
            </div>
        </div>
    );
}
