// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {fireEvent, render, screen} from '@testing-library/react';

import {RHSTab} from 'types/rhs';

import RHSTabStrip from './rhs_tab_strip';

const assignedTab: RHSTab = {kind: 'assigned', name: 'Assigned to me'};
const inProgressTab: RHSTab = {kind: 'category', name: 'In Progress', key: 'indeterminate'};

describe('components/rhs/rhs_tab_strip', () => {
    test('does not paint overflow fades when tabs fit', () => {
        const {container} = render(
            <RHSTabStrip
                tabs={[assignedTab, inProgressTab]}
                selectedTab={assignedTab}
                onSelect={jest.fn()}
            />,
        );

        const wrap = container.querySelector('.jira-rhs-tab-strip-wrap');
        expect(wrap).not.toHaveClass('jira-rhs-tab-strip-wrap--fade-right');
        expect(wrap).not.toHaveClass('jira-rhs-tab-strip-wrap--fade-left');
    });

    test('arrow right selects the next tab', () => {
        const onSelect = jest.fn();
        render(
            <RHSTabStrip
                tabs={[assignedTab, inProgressTab]}
                selectedTab={assignedTab}
                onSelect={onSelect}
            />,
        );

        fireEvent.keyDown(screen.getByRole('tab', {name: assignedTab.name}), {key: 'ArrowRight'});
        expect(onSelect).toHaveBeenCalledWith(inProgressTab);
    });
});
