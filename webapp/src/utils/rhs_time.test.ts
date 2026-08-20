// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {formatRHSRelativeTime} from './rhs_time';

const nowMs = Date.parse('2026-01-02T01:00:00Z');

describe('rhs_time', () => {
    test('formatRHSRelativeTime returns empty string for empty or invalid input', () => {
        expect(formatRHSRelativeTime('', nowMs)).toBe('');
        expect(formatRHSRelativeTime('not-a-date', nowMs)).toBe('');
    });

    test('formatRHSRelativeTime returns just now under one minute', () => {
        expect(formatRHSRelativeTime('2026-01-02T00:59:30Z', nowMs)).toBe('just now');
    });

    test('formatRHSRelativeTime returns Nm ago under one hour', () => {
        expect(formatRHSRelativeTime('2026-01-02T00:15:00Z', nowMs)).toBe('45m ago');
    });

    test('formatRHSRelativeTime returns Nh ago under one day', () => {
        expect(formatRHSRelativeTime('2026-01-01T20:00:00Z', nowMs)).toBe('5h ago');
    });

    test('formatRHSRelativeTime returns an ISO date after one week', () => {
        expect(formatRHSRelativeTime('2025-12-20T00:00:00Z', nowMs)).toBe('2025-12-20');
    });
});
