// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {isMinimumServerVersion} from 'mattermost-redux/utils/helpers';

const userAgent = window.navigator.userAgent;

export function isDesktopApp() {
    return userAgent.indexOf('Mattermost') !== -1 && userAgent.indexOf('Electron') !== -1;
}

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
