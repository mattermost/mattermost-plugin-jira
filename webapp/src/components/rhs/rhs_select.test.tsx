// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {fireEvent, render, screen} from '@testing-library/react';

import RHSSelect from './rhs_select';

describe('components/rhs/rhs_select', () => {
    test('opens a themed listbox and reports the chosen value', () => {
        const onChange = jest.fn();
        render(
            <RHSSelect
                ariaLabel='Sort'
                testId='rhs-sort'
                value='updated'
                options={[
                    {value: 'updated', label: 'Updated'},
                    {value: 'created', label: 'Created'},
                ]}
                onChange={onChange}
            />,
        );

        expect(screen.queryByRole('listbox')).toBeNull();
        fireEvent.click(screen.getByTestId('rhs-sort'));
        fireEvent.click(screen.getByRole('option', {name: 'Created'}));
        expect(onChange).toHaveBeenCalledWith('created');
    });
});
