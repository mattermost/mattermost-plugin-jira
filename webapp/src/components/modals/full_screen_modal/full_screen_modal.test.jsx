// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {fireEvent, render} from '@testing-library/react';
import {IntlProvider} from 'react-intl';

import FullScreenModal from './full_screen_modal.jsx';

const renderWithIntl = (ui) => {
    return render(
        <IntlProvider locale='en'>
            {ui}
        </IntlProvider>,
    );
};

describe('components/widgets/modals/FullScreenModal', () => {
    test('showing content', () => {
        const {container} = renderWithIntl(
            <FullScreenModal
                show={true}
                onClose={jest.fn()}
            >
                {'test'}
            </FullScreenModal>,
        );

        expect(container.querySelector('.FullScreenModal')).toBeInTheDocument();
        expect(container.textContent).toContain('test');
    });

    test('not showing content', () => {
        const {container} = renderWithIntl(
            <FullScreenModal
                show={false}
                onClose={jest.fn()}
            >
                {'test'}
            </FullScreenModal>,
        );

        expect(container.querySelector('.FullScreenModal')).not.toBeInTheDocument();
    });

    test('close on close icon click', () => {
        const close = jest.fn();
        const {container} = renderWithIntl(
            <FullScreenModal
                show={true}
                onClose={close}
            >
                {'test'}
            </FullScreenModal>,
        );

        expect(close).not.toHaveBeenCalled();
        const closeIcon = container.querySelector('.close-x');
        fireEvent.click(closeIcon);
        expect(close).toHaveBeenCalled();
    });

    test('close on esc keypress', () => {
        const close = jest.fn();
        renderWithIntl(
            <FullScreenModal
                show={true}
                onClose={close}
            >
                {'test'}
            </FullScreenModal>,
        );

        expect(close).not.toHaveBeenCalled();
        const event = new KeyboardEvent('keydown', {key: 'Escape'});
        document.dispatchEvent(event);
        expect(close).toHaveBeenCalled();
    });
});
