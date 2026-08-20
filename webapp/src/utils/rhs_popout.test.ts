// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {isRHSPopoutPathname} from './rhs_popout';

describe('rhs_popout', () => {
    test('isRHSPopoutPathname is true for the 11.3 popout path shape', () => {
        expect(isRHSPopoutPathname('/_popout/rhs/team/channel/plugin/jira')).toBe(true);
    });

    test('isRHSPopoutPathname is true for the 11.6 popout path shape', () => {
        expect(isRHSPopoutPathname('/_popout/rhs/team/plugin/jira')).toBe(true);
    });

    test('isRHSPopoutPathname is false for a normal team channel path', () => {
        expect(isRHSPopoutPathname('/team/channel')).toBe(false);
    });

    test('isRHSPopoutPathname is false when _popout appears later in the path', () => {
        expect(isRHSPopoutPathname('/team/_popout/nope')).toBe(false);
    });
});
