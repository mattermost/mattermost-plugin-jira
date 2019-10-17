// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {isMinimumServerVersion} from 'mattermost-redux/utils/helpers';

const userAgent = window.navigator.userAgent;

export function isDesktopApp() {
    return userAgent.indexOf('Mattermost') !== -1 && userAgent.indexOf('Electron') !== -1;
}

// Desktop app user agent examples:
// Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Mattermost/4.3.0-develop Chrome/73.0.3683.121 Electron/5.0.10 Safari/537.36'
// Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Mattermost/4.2.3 Chrome/61.0.3163.100 Electron/2.0.12 Safari/537.36
export function getDesktopAppVersion() {
    if (!isDesktopApp()) {
        return null;
    }

    const beginIndex = userAgent.indexOf('Mattermost') + 'Mattermost/'.length;
    const developDashIndex = userAgent.substring(beginIndex).indexOf('-');
    const spaceIndex = userAgent.substring(beginIndex).indexOf(' ');
    const first = Math.min(developDashIndex, spaceIndex);

    return userAgent.substring(beginIndex, beginIndex + first);
}

export function isMinimumDesktopAppVersion(minMajorVersion, minMinorVersion, minDotVersion) {
    const currentVersion = getDesktopAppVersion();
    if (!currentVersion) {
        return false;
    }

    return isMinimumServerVersion(currentVersion, minMajorVersion, minMinorVersion, minDotVersion);
}
