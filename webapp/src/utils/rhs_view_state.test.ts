// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {RHSViewState} from 'types/rhs';

import {loadRHSViewState, saveRHSViewState, validateRHSViewState} from './rhs_view_state';

describe('rhs_view_state', () => {
    afterEach(() => {
        localStorage.clear();
        jest.restoreAllMocks();
    });

    test('rhs_view_state round-trips instance tab sort', () => {
        const view: RHSViewState = {
            instance: 'https://example.atlassian.net',
            tab: {kind: 'category', name: 'In Progress', key: 'indeterminate'},
            sort: 'created',
        };

        saveRHSViewState('u1', view);
        expect(loadRHSViewState('u1')).toEqual(view);
    });

    test('rhs_view_state isolates keys by user id', () => {
        saveRHSViewState('u1', {
            instance: 'https://example.atlassian.net',
            tab: {kind: 'assigned', name: 'Assigned'},
            sort: 'updated',
        });

        expect(loadRHSViewState('u2')).toBeNull();
    });

    test('loadRHSViewState returns null when localStorage throws', () => {
        jest.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
            throw new Error('quota');
        });

        expect(loadRHSViewState('u1')).toBeNull();
    });

    test('saveRHSViewState swallows a localStorage throw', () => {
        jest.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
            throw new Error('quota');
        });

        expect(() => {
            saveRHSViewState('u1', {
                instance: 'https://example.atlassian.net',
                tab: {kind: 'assigned', name: 'Assigned'},
                sort: 'updated',
            });
        }).not.toThrow();
    });

    test('validateRHSViewState rewrites a stored assigned label to Assigned to me', () => {
        expect(validateRHSViewState({
            instance: 'x',
            tab: {kind: 'assigned', name: 'Assigned'},
            sort: 'updated',
        })).toEqual({
            instance: 'x',
            tab: {kind: 'assigned', name: 'Assigned to me'},
            sort: 'updated',
        });
    });

    test('validateRHSViewState rejects an unknown tab kind', () => {
        expect(validateRHSViewState({
            instance: 'x',
            tab: {kind: 'nope', name: 'X'},
            sort: 'updated',
        })).toBeNull();
    });

    test('validateRHSViewState rejects a sort outside updated or created', () => {
        expect(validateRHSViewState({
            instance: 'x',
            tab: {kind: 'assigned', name: 'Assigned'},
            sort: 'priority',
        })).toBeNull();
    });

    test('validateRHSViewState rejects a category tab without key', () => {
        expect(validateRHSViewState({
            instance: 'x',
            tab: {kind: 'category', name: 'In Progress'},
            sort: 'updated',
        })).toBeNull();
    });

    test('validateRHSViewState rejects a status tab without id', () => {
        expect(validateRHSViewState({
            instance: 'x',
            tab: {kind: 'status', name: 'Blocked'},
            sort: 'updated',
        })).toBeNull();
    });
});
