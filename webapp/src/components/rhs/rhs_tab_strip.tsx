// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {RHSTab} from 'types/model';
import {rhsTabsEqual} from 'utils/rhs_resolve';

export type Props = {
    tabs: RHSTab[];
    selectedTab: RHSTab;
    onSelect: (tab: RHSTab) => void;
};

export default function RHSTabStrip(props: Props): JSX.Element {
    return (
        <div
            className='jira-rhs-tab-strip-wrap'
            data-testid='rhs-tab-strip'
        >
            <div
                className='jira-rhs-tab-strip'
                role='tablist'
            >
                {props.tabs.map((item) => {
                    const selected = rhsTabsEqual(item, props.selectedTab);
                    return (
                        <button
                            key={item.kind + ':' + (item.key || '') + ':' + (item.id || '')}
                            className={selected ? 'jira-rhs-tab jira-rhs-tab--selected' : 'jira-rhs-tab'}
                            role='tab'
                            aria-selected={selected}
                            type='button'
                            onClick={() => props.onSelect(item)}
                        >
                            {item.name}
                        </button>
                    );
                })}
            </div>
        </div>
    );
}
