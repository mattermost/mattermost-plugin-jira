// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export function isRHSPopoutPathname(pathname: string): boolean {
    return pathname.indexOf('/_popout/') === 0;
}
