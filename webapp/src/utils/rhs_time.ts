// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const minuteMs = 60 * 1000;
const hourMs = 60 * minuteMs;
const dayMs = 24 * hourMs;
const weekMs = 7 * dayMs;

export function formatRHSRelativeTime(iso: string, nowMs: number): string {
    if (!iso) {
        return '';
    }
    const then = Date.parse(iso);
    if (Number.isNaN(then)) {
        return '';
    }
    const delta = Math.max(0, nowMs - then);
    if (delta < minuteMs) {
        return 'just now';
    }
    if (delta < hourMs) {
        return Math.floor(delta / minuteMs) + 'm ago';
    }
    if (delta < dayMs) {
        return Math.floor(delta / hourMs) + 'h ago';
    }
    if (delta < weekMs) {
        return Math.floor(delta / dayMs) + 'd ago';
    }
    return iso.slice(0, 10);
}
