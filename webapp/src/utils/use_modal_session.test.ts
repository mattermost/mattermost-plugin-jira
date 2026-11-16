// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {renderHook} from '@testing-library/react';

import useModalSession from './use_modal_session';

type Props = {visible: boolean; postId?: string};

describe('useModalSession', () => {
    it('keeps the last open props while closing and starts a new session on reopen', () => {
        const {result, rerender} = renderHook((props: Props) => useModalSession(props), {
            initialProps: {visible: true, postId: 'first'},
        });
        const firstKey = result.current.key;

        rerender({visible: false});
        expect(result.current.key).toBe(firstKey);
        expect(result.current.props).toEqual({visible: false, postId: 'first'});

        rerender({visible: true, postId: 'second'});
        expect(result.current.key).not.toBe(firstKey);
        expect(result.current.props).toEqual({visible: true, postId: 'second'});
    });
});
