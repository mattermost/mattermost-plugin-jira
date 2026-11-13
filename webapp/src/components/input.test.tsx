// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {
    act,
    fireEvent,
    render,
    screen,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import Input from './input';

const dayInCurrentMonth = (day: number) => {
    const now = new Date();
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(day)}`;
};

describe('components/Input', () => {
    test('should pick a date from the calendar', async () => {
        const onChange = jest.fn();
        render(
            <Input
                id='duedate'
                label='Due Date'
                type='date'
                value='2026-03-10'
                onChange={onChange}
            />,
        );

        const field = screen.getByRole('group', {name: 'Due Date'});
        await userEvent.click(field.querySelector('[role="button"]')!);
        await userEvent.click(screen.getByRole('button', {name: dayInCurrentMonth(12)}));

        expect(onChange).toHaveBeenCalledWith('duedate', dayInCurrentMonth(12));
    });

    test('should combine the picked date and typed time for datetime fields', async () => {
        const onChange = jest.fn();
        render(
            <Input
                id='start'
                label='Start'
                type='datetime-local'
                value='2026-03-10T09:30'
                onChange={onChange}
            />,
        );

        await userEvent.click(screen.getByRole('group', {name: 'Start'}).querySelector('[role="button"]')!);
        await userEvent.click(screen.getByRole('button', {name: dayInCurrentMonth(12)}));
        expect(onChange).toHaveBeenLastCalledWith('start', `${dayInCurrentMonth(12)}T09:30`);

        fireEvent.change(screen.getByLabelText('Time'), {target: {value: '14:15'}});
        expect(onChange).toHaveBeenLastCalledWith('start', '2026-03-10T14:15');
    });

    test('should disable the time until a date is picked', () => {
        render(
            <Input
                id='start'
                label='Start'
                type='datetime-local'
                value=''
            />,
        );

        expect(screen.getByLabelText('Time')).toBeDisabled();
    });

    test('should report a missing required date below the field', () => {
        const ref = React.createRef<Input>();
        render(
            <Input
                ref={ref}
                id='duedate'
                label='Due Date'
                type='date'
                required={true}
                value=''
            />,
        );

        act(() => {
            ref.current?.isValid();
        });

        expect(screen.getByRole('group', {name: 'Due Date'})).toHaveAccessibleDescription('This field is required.');
    });

    test('should clear a picked date', async () => {
        const onChange = jest.fn();
        const {rerender} = render(
            <Input
                id='duedate'
                label='Due Date'
                type='date'
                value='2026-03-10'
                onChange={onChange}
            />,
        );

        await userEvent.click(screen.getByRole('button', {name: 'Clear Due Date'}));
        expect(onChange).toHaveBeenCalledWith('duedate', '');

        rerender(
            <Input
                id='duedate'
                label='Due Date'
                type='date'
                value=''
                onChange={onChange}
            />,
        );
        expect(screen.queryByRole('button', {name: 'Clear Due Date'})).not.toBeInTheDocument();
    });

    test('should not let a read-only date be changed', () => {
        render(
            <Input
                id='start'
                label='Start'
                type='datetime-local'
                value='2026-03-10T09:30'
                readOnly={true}
            />,
        );

        expect(screen.queryByRole('button', {name: 'Clear Start'})).not.toBeInTheDocument();
        expect(screen.getByLabelText('Time')).toBeDisabled();
    });
});
