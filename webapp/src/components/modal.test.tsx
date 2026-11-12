// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import {Select} from '@mattermost/compass-ui/components/select';

import ConfirmModal from 'components/confirm_modal';
import Input from 'components/input';

import Modal from './modal';

describe('components/Modal', () => {
    const swallowEscape = (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
            e.preventDefault();
        }
    };

    beforeEach(() => {
        document.addEventListener('keydown', swallowEscape);
    });

    afterEach(() => {
        document.removeEventListener('keydown', swallowEscape);
    });

    test('should close on Escape even when a host listener prevents it first', async () => {
        const onClose = jest.fn();
        render(
            <Modal
                title='Title'
                onClose={onClose}
            >
                {'Body'}
            </Modal>,
        );

        await userEvent.keyboard('{Escape}');

        expect(onClose).toHaveBeenCalledTimes(1);
    });

    test('should close the topmost modal on Escape when focus has fallen back to the body', async () => {
        const onClose = jest.fn();
        const onCancel = jest.fn();
        const renderStack = (show: boolean) => (
            <Modal
                title='Title'
                onClose={onClose}
            >
                <ConfirmModal
                    show={show}
                    title='Discard'
                    message='Discard changes?'
                    confirmButtonText='Discard'
                    onConfirm={jest.fn()}
                    onCancel={onCancel}
                />
            </Modal>
        );
        const {rerender} = render(renderStack(false));

        (document.activeElement as HTMLElement).blur();
        await userEvent.keyboard('{Escape}');
        expect(onClose).toHaveBeenCalledTimes(1);

        rerender(renderStack(true));
        await waitFor(() => expect(screen.getByRole('button', {name: 'Discard'})).toHaveFocus());
        (document.activeElement as HTMLElement).blur();
        await userEvent.keyboard('{Escape}');

        expect(onCancel).toHaveBeenCalledTimes(1);
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    test('should leave Escape to an open menu inside the dialog', async () => {
        const onClose = jest.fn();
        render(
            <Modal
                title='Title'
                onClose={onClose}
            >
                <Select
                    id='fruit'
                    label='Fruit'
                    options={[{value: 'apple', label: 'Apple'}]}
                    value={null}
                    onChange={jest.fn()}
                />
            </Modal>,
        );

        await userEvent.click(screen.getByRole('combobox', {name: 'Fruit'}));
        expect(screen.getByRole('listbox')).toBeInTheDocument();

        await userEvent.keyboard('{Escape}');

        expect(onClose).not.toHaveBeenCalled();
    });

    test('should leave Escape to an open date picker inside the dialog', async () => {
        const onClose = jest.fn();
        render(
            <Modal
                title='Title'
                onClose={onClose}
            >
                <Input
                    id='duedate'
                    label='Due Date'
                    type='date'
                    value=''
                />
            </Modal>,
        );

        await userEvent.click(screen.getByRole('group', {name: 'Due Date'}).querySelector('[role="button"]')!);
        expect(screen.getByRole('dialog', {name: 'Date picker'})).toBeInTheDocument();

        await userEvent.keyboard('{Escape}');

        expect(onClose).not.toHaveBeenCalled();
        await waitFor(() => expect(screen.queryByRole('dialog', {name: 'Date picker'})).not.toBeInTheDocument());
    });

    test('should only close the topmost of stacked modals', async () => {
        const onClose = jest.fn();
        const onCancel = jest.fn();
        const renderStack = (show: boolean) => (
            <Modal
                title='Title'
                onClose={onClose}
            >
                <ConfirmModal
                    show={show}
                    title='Discard'
                    message='Discard changes?'
                    confirmButtonText='Discard'
                    onConfirm={jest.fn()}
                    onCancel={onCancel}
                />
            </Modal>
        );
        const {rerender} = render(renderStack(false));
        rerender(renderStack(true));

        await waitFor(() => expect(screen.getByRole('button', {name: 'Discard'})).toHaveFocus());
        await userEvent.keyboard('{Escape}');

        expect(onCancel).toHaveBeenCalledTimes(1);
        expect(onClose).not.toHaveBeenCalled();
    });

    test('should show a tooltip on the back button', async () => {
        render(
            <Modal
                title='Title'
                showBackButton={true}
                onBack={jest.fn()}
            >
                {'Body'}
            </Modal>,
        );

        const back = screen.getByRole('button', {name: 'Back'});
        await userEvent.hover(back);
        expect(screen.getByTestId('jira-tooltip')).toHaveTextContent('Back');

        await userEvent.unhover(back);
        expect(screen.queryByTestId('jira-tooltip')).not.toBeInTheDocument();
    });
});
