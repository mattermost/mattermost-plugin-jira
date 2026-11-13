// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import SelectSetting from './select_setting';

const options = [
    {label: 'Bug', value: '1'},
    {label: 'Task', value: '2'},
];

describe('components/SelectSetting', () => {
    test('should report the picked value with the field name', async () => {
        const onChange = jest.fn();
        render(
            <SelectSetting
                name='issue_type'
                label='Issue Type'
                options={options}
                onChange={onChange}
            />,
        );

        await userEvent.click(screen.getByRole('combobox', {name: 'Issue Type'}));
        await userEvent.click(await screen.findByRole('option', {name: 'Task'}));

        expect(onChange).toHaveBeenCalledWith('issue_type', '2');
    });

    test('should report every picked value in multi mode', async () => {
        const onChange = jest.fn();
        render(
            <SelectSetting
                name='events'
                label='Events'
                options={options}
                value={[options[0]]}
                onChange={onChange}
                isMulti={true}
            />,
        );

        await userEvent.click(screen.getByRole('combobox', {name: 'Events'}));
        await userEvent.click(await screen.findByRole('option', {name: 'Task'}));

        expect(onChange).toHaveBeenCalledWith('events', ['1', '2']);
    });

    test('should flag a missing required value when validated', async () => {
        let validate: () => boolean = () => true;
        render(
            <SelectSetting
                name='project'
                label='Project'
                options={options}
                required={true}
                addValidate={(isValid) => {
                    validate = isValid;
                }}
            />,
        );

        let valid = true;
        act(() => {
            valid = validate();
        });

        expect(valid).toBe(false);
        expect(screen.getByText('This field is required.')).toBeInTheDocument();
    });

    test('should cap large static option lists and filter them by the typed text', async () => {
        const manyOptions = Array.from({length: 150}, (_, i) => ({label: `Project ${i}`, value: `P${i}`}));
        render(
            <SelectSetting
                name='projects'
                label='Project'
                options={manyOptions}
                limitOptions={true}
            />,
        );

        const input = screen.getByRole('combobox', {name: 'Project'});
        await userEvent.click(input);
        expect(await screen.findAllByRole('option')).toHaveLength(100);

        await userEvent.type(input, 'Project 14');
        expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual([
            'Project 14',
            ...Array.from({length: 10}, (_, i) => `Project 14${i}`),
        ]);
    });

    test('should describe a required error even without a field name', () => {
        const ref = React.createRef<SelectSetting>();
        render(
            <SelectSetting
                ref={ref}
                aria-label='Values'
                options={options}
                required={true}
            />,
        );

        act(() => {
            ref.current?.isValid();
        });

        expect(screen.getByRole('combobox', {name: 'Values'})).toHaveAccessibleDescription('This field is required.');
    });
});
