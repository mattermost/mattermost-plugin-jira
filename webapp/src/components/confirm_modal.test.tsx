// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import ConfirmModal, {Props} from './confirm_modal';

describe('components/ConfirmModal', () => {
    const baseProps: Props = {
        show: true,
        title: 'Subscription',
        message: 'Are you sure to delete the subscription?',
        confirmButtonText: 'Delete',
        destructive: true,
        onConfirm: jest.fn(),
        onCancel: jest.fn(),
    };

    test('should render a dialog named by its title with focus on the confirm button', () => {
        render(<ConfirmModal {...baseProps}/>);

        expect(screen.getByRole('dialog', {name: 'Subscription'})).toHaveTextContent('Are you sure to delete the subscription?');
        expect(screen.getByRole('button', {name: 'Delete'})).toHaveFocus();
    });

    test('should pass the checkbox state to the confirm callback', async () => {
        const onConfirm = jest.fn();
        render(
            <ConfirmModal
                {...baseProps}
                showCheckbox={true}
                checkboxText='Do not ask again'
                onConfirm={onConfirm}
            />,
        );

        await userEvent.click(screen.getByRole('checkbox', {name: 'Do not ask again'}));
        await userEvent.click(screen.getByRole('button', {name: 'Delete'}));

        expect(onConfirm).toHaveBeenCalledWith(true);
    });

    test('should cancel from the cancel button and from Escape', async () => {
        const onCancel = jest.fn();
        render(
            <ConfirmModal
                {...baseProps}
                onCancel={onCancel}
            />,
        );

        await userEvent.click(screen.getByRole('button', {name: 'Cancel'}));
        await userEvent.keyboard('{Escape}');

        expect(onCancel).toHaveBeenCalledTimes(2);
        expect(onCancel).toHaveBeenCalledWith(false);
    });

    test('should not render a cancel button when hidden', () => {
        render(
            <ConfirmModal
                {...baseProps}
                hideCancel={true}
            />,
        );

        expect(screen.queryByRole('button', {name: 'Cancel'})).not.toBeInTheDocument();
    });
});
